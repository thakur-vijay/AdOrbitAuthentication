//go:build integration

package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/adorbit/authentication/internal/database"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/thakur-vijay/adorbit-go/mailservice"
)

func postgresFixture(t testing.TB) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required for integration tests; start PostgreSQL or configure the CI service container")
	}

	adminDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open TEST_DATABASE_URL: %v", err)
	}
	adminDB.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := adminDB.PingContext(ctx); err != nil {
		_ = adminDB.Close()
		t.Fatalf("connect to PostgreSQL at TEST_DATABASE_URL: %v", err)
	}

	schema := "it_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := adminDB.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		_ = adminDB.Close()
		t.Fatalf("create isolated PostgreSQL schema: %v", err)
	}
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		_, _ = adminDB.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE")
		_ = adminDB.Close()
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	query := parsedURL.Query()
	query.Set("search_path", schema+",public")
	parsedURL.RawQuery = query.Encode()
	db, err := sql.Open("pgx", parsedURL.String())
	if err != nil {
		_, _ = adminDB.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE")
		_ = adminDB.Close()
		t.Fatalf("open isolated PostgreSQL connection: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		_, _ = adminDB.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE")
		_ = adminDB.Close()
		t.Fatalf("connect to isolated PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := adminDB.ExecContext(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop isolated PostgreSQL schema: %v", err)
		}
		if err := adminDB.Close(); err != nil {
			t.Errorf("close PostgreSQL admin connection: %v", err)
		}
	})
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get integration test working directory: %v", err)
	}
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("change to project root for actual migrations: %v", err)
	}
	migrationErr := database.Migrate(db, parsedURL.String())
	workingDirectoryErr := os.Chdir(workingDirectory)
	if migrationErr != nil {
		t.Fatalf("database.Migrate() failed against isolated PostgreSQL schema: %v", migrationErr)
	}
	if workingDirectoryErr != nil {
		t.Fatalf("restore integration test working directory: %v", workingDirectoryErr)
	}
	return db
}

func insertCredential(t testing.TB, db *sql.DB, userID, email string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO credentials (user_id, email, password_hash) VALUES ($1, $2, $3)`, userID, email, "test-hash"); err != nil {
		t.Fatalf("insert test credential: %v", err)
	}
}

type usersServiceHarness struct {
	t         testing.TB
	conn      *amqp.Connection
	createCh  *amqp.Channel
	fetchCh   *amqp.Channel
	usersMu   sync.Mutex
	users     map[string]map[string]any
	heldOnce  sync.Once
	held      chan struct{}
	waitGroup sync.WaitGroup
}

func startUsersServiceHarness(t testing.TB) *usersServiceHarness {
	t.Helper()
	brokerURL := os.Getenv("TEST_RABBITMQ_URL")
	if brokerURL == "" {
		t.Fatal("TEST_RABBITMQ_URL is required for integration tests; start RabbitMQ or configure the CI service container")
	}
	conn, err := amqp.Dial(brokerURL)
	if err != nil {
		t.Fatalf("connect to RabbitMQ at TEST_RABBITMQ_URL: %v", err)
	}
	createCh, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		t.Fatalf("open RabbitMQ create consumer channel: %v", err)
	}
	fetchCh, err := conn.Channel()
	if err != nil {
		_ = createCh.Close()
		_ = conn.Close()
		t.Fatalf("open RabbitMQ fetch consumer channel: %v", err)
	}
	createQueue, err := createCh.QueueDeclare("users.create", true, false, false, false, nil)
	if err != nil {
		_ = fetchCh.Close()
		_ = createCh.Close()
		_ = conn.Close()
		t.Fatalf("declare users.create queue: %v", err)
	}
	fetchQueue, err := fetchCh.QueueDeclare("users.fetch", true, false, false, false, nil)
	if err != nil {
		_ = fetchCh.Close()
		_ = createCh.Close()
		_ = conn.Close()
		t.Fatalf("declare users.fetch queue: %v", err)
	}
	createMessages, err := createCh.Consume(createQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		_ = fetchCh.Close()
		_ = createCh.Close()
		_ = conn.Close()
		t.Fatalf("consume users.create queue: %v", err)
	}
	fetchMessages, err := fetchCh.Consume(fetchQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		_ = fetchCh.Close()
		_ = createCh.Close()
		_ = conn.Close()
		t.Fatalf("consume users.fetch queue: %v", err)
	}
	harness := &usersServiceHarness{
		t:        t,
		conn:     conn,
		createCh: createCh,
		fetchCh:  fetchCh,
		users:    make(map[string]map[string]any),
		held:     make(chan struct{}),
	}
	harness.waitGroup.Add(2)
	go harness.consumeCreates(createMessages)
	go harness.consumeFetches(fetchMessages)
	t.Cleanup(func() {
		_ = createCh.Close()
		_ = fetchCh.Close()
		finished := make(chan struct{})
		go func() {
			harness.waitGroup.Wait()
			close(finished)
		}()
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Error("RabbitMQ test consumers did not stop after channel closure")
		}
		cleanupChannel, err := conn.Channel()
		if err != nil {
			t.Errorf("open RabbitMQ cleanup channel: %v", err)
		} else {
			if _, err := cleanupChannel.QueueDelete("users.create", false, false, false); err != nil {
				t.Errorf("delete test users.create queue: %v", err)
			}
			if _, err := cleanupChannel.QueueDelete("users.fetch", false, false, false); err != nil {
				t.Errorf("delete test users.fetch queue: %v", err)
			}
			if err := cleanupChannel.Close(); err != nil {
				t.Errorf("close RabbitMQ cleanup channel: %v", err)
			}
		}
		if err := conn.Close(); err != nil {
			t.Errorf("close RabbitMQ test service connection: %v", err)
		}
	})
	return harness
}

func (h *usersServiceHarness) consumeCreates(messages <-chan amqp.Delivery) {
	defer h.waitGroup.Done()
	for message := range messages {
		var request struct {
			UserID   string `json:"user_id"`
			FullName string `json:"full_name"`
			Email    string `json:"email"`
			Role     string `json:"role"`
		}
		if err := json.Unmarshal(message.Body, &request); err != nil {
			h.nack(message)
			continue
		}
		if request.FullName == "HOLD_RESPONSE" {
			h.heldOnce.Do(func() { close(h.held) })
			h.ack(message)
			continue
		}
		response := map[string]any{
			"user_id":    request.UserID,
			"full_name":  request.FullName,
			"email":      request.Email,
			"role":       request.Role,
			"created_at": time.Now().UTC(),
			"updated_at": time.Now().UTC(),
		}
		h.usersMu.Lock()
		_, exists := h.users[request.UserID]
		if !exists {
			h.users[request.UserID] = response
		}
		h.usersMu.Unlock()
		if exists {
			response = map[string]any{"code": "USER_ALREADY_EXISTS", "message": "user already exists"}
		} else if request.FullName == "UNEXPECTED_ID" {
			response["user_id"] = request.UserID + "-unexpected"
		} else if request.FullName == "MALFORMED_RESPONSE" {
			h.publishResponse(h.createCh, message, []byte("{"))
			h.ack(message)
			continue
		}
		body, err := json.Marshal(response)
		if err != nil {
			h.t.Errorf("marshal test create-user response: %v", err)
			h.nack(message)
			continue
		}
		if request.FullName == "MISMATCH_CORRELATION" {
			mismatched := message
			mismatched.CorrelationId += "-wrong"
			h.publishResponse(h.createCh, mismatched, body)
		}
		h.publishResponse(h.createCh, message, body)
		h.ack(message)
	}
}

func (h *usersServiceHarness) consumeFetches(messages <-chan amqp.Delivery) {
	defer h.waitGroup.Done()
	for message := range messages {
		var request struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		}
		if err := json.Unmarshal(message.Body, &request); err != nil {
			h.nack(message)
			continue
		}
		h.usersMu.Lock()
		user, exists := h.users[request.UserID]
		var response map[string]any
		if exists {
			response = make(map[string]any, len(user))
			for key, value := range user {
				response[key] = value
			}
		} else {
			response = map[string]any{"code": "USER_NOT_FOUND", "message": "user not found"}
		}
		h.usersMu.Unlock()
		body, err := json.Marshal(response)
		if err != nil {
			h.t.Errorf("marshal test fetch-user response: %v", err)
			h.nack(message)
			continue
		}
		h.publishResponse(h.fetchCh, message, body)
		h.ack(message)
	}
}

func (h *usersServiceHarness) publishResponse(channel *amqp.Channel, request amqp.Delivery, body []byte) {
	err := channel.Publish("", request.ReplyTo, false, false, amqp.Publishing{
		ContentType:   "application/json",
		CorrelationId: request.CorrelationId,
		Body:          body,
	})
	if err != nil {
		h.t.Errorf("publish test Users Service reply: %v", err)
	}
}

func (h *usersServiceHarness) ack(message amqp.Delivery) {
	if err := message.Ack(false); err != nil {
		h.t.Errorf("acknowledge test Users Service request: %v", err)
	}
}

func (h *usersServiceHarness) nack(message amqp.Delivery) {
	if err := message.Nack(false, false); err != nil {
		h.t.Errorf("reject malformed test Users Service request: %v", err)
	}
}

type captureMailer struct {
	messages chan mailservice.Message
	err      error
}

func newCaptureMailer() *captureMailer {
	return &captureMailer{messages: make(chan mailservice.Message, 16)}
}

func (m *captureMailer) Send(ctx context.Context, message mailservice.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.err != nil {
		return m.err
	}
	select {
	case m.messages <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func requireEnv(t testing.TB, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Fatalf("%s is required for integration tests", key)
	}
	return value
}

func migrationFiles(t testing.TB) []string {
	t.Helper()
	directory := filepath.Join("..", "..", "internal", "database", "migrations")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read migration directory: %v", err)
	}
	var names []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names
}

func applyDownMigrations(t testing.TB, db *sql.DB) {
	t.Helper()
	names := migrationFiles(t)
	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]
		if !strings.HasSuffix(name, ".down.sql") {
			continue
		}
		contents, err := os.ReadFile(filepath.Join("..", "..", "internal", "database", "migrations", name))
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		if _, err := db.Exec(string(contents)); err != nil {
			t.Fatalf("apply down migration %s: %v", name, err)
		}
	}
}

func assertNoTable(t testing.TB, db *sql.DB, name string) {
	t.Helper()
	var found bool
	if err := db.QueryRow(`SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL`, name).Scan(&found); err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	if found {
		t.Fatalf("table %s remains after down migrations", name)
	}
}

func mustUUID(t testing.TB) string {
	t.Helper()
	return uuid.NewString()
}
