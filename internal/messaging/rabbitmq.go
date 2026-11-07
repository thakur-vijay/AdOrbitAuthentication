package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/thakur-vijay/adorbit-go/apperrors"
)

type RabbitMQ struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	return &RabbitMQ{
		Connection: conn,
		Channel:    channel,
	}, nil
}

func (r *RabbitMQ) Close() error {
	if r == nil {
		return nil
	}

	var closeErr error

	if r.Channel != nil {
		if err := r.Channel.Close(); err != nil {
			closeErr = err
		}
	}

	if r.Connection != nil {
		if err := r.Connection.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}

	return closeErr
}

func (r *RabbitMQ) DeclareQueue(name string) (amqp.Queue, error) {
	return r.Channel.QueueDeclare(
		name,
		true,
		false,
		false,
		false,
		nil,
	)
}

type CreateUserRequest struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type FetchUserRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type UserResponse struct {
	UserID    string    `json:"user_id,omitempty"`
	FullName  string    `json:"full_name,omitempty"`
	Email     string    `json:"email,omitempty"`
	Role      string    `json:"role,omitempty"`
	Code      string    `json:"code,omitempty"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *RabbitMQ) CreateUser(
	ctx context.Context,
	request CreateUserRequest,
) (*UserResponse, error) {
	if r == nil || r.Connection == nil {
		return nil, apperrors.NewInternal(
			"RabbitMQ connection is not initialized",
			nil,
		)
	}

	if err := ctx.Err(); err != nil {
		return nil, apperrors.NewRequestTimeout(
			"Create user request was cancelled",
			err,
		)
	}

	if request.UserID == "" {
		return nil, apperrors.NewBadRequest(
			"User ID is required",
			nil,
		)
	}

	if request.Role == "" {
		return nil, apperrors.NewBadRequest(
			"Role is required",
			nil,
		)
	}

	channel, err := r.Connection.Channel()
	if err != nil {
		return nil, apperrors.NewInternal(
			"Failed to open RabbitMQ channel",
			err,
		)
	}
	defer channel.Close()

	queue, err := channel.QueueDeclare(
		"users.create",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, apperrors.NewInternal(
			"Failed to declare users queue",
			err,
		)
	}

	replyQueue, err := channel.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		nil,
	)
	if err != nil {
		return nil, apperrors.NewInternal(
			"Failed to declare reply queue",
			err,
		)
	}

	messages, err := channel.Consume(
		replyQueue.Name,
		"",
		true,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, apperrors.NewInternal(
			"Failed to consume reply queue",
			err,
		)
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, apperrors.NewInternal(
			"Failed to encode create user request",
			err,
		)
	}

	correlationID := fmt.Sprintf(
		"%s-%d",
		request.UserID,
		time.Now().UnixNano(),
	)

	err = channel.PublishWithContext(
		ctx,
		"",
		queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			ReplyTo:       replyQueue.Name,
			Body:          body,
			DeliveryMode:  amqp.Persistent,
		},
	)
	if err != nil {
		return nil, apperrors.NewInternal(
			"Failed to publish create user request",
			err,
		)
	}

	for {
		select {
		case <-ctx.Done():
			return nil, apperrors.NewRequestTimeout(
				"Timed out waiting for user creation response",
				ctx.Err(),
			)

		case message, ok := <-messages:
			if !ok {
				return nil, apperrors.NewInternal(
					"RabbitMQ reply channel was closed",
					nil,
				)
			}

			if message.CorrelationId != correlationID {
				continue
			}

			var response UserResponse

			if err := json.Unmarshal(message.Body, &response); err != nil {
				return nil, apperrors.NewInternal(
					"Invalid response from users service",
					err,
				)
			}

			if response.Code != "" {
				switch response.Code {
				case "USER_ALREADY_EXISTS":
					return nil, apperrors.NewConflict(
						response.Message,
						nil,
					)

				default:
					return nil, apperrors.NewInternal(
						response.Message,
						nil,
					)
				}
			}

			if response.UserID == "" {
				return nil, apperrors.NewInternal(
					"Users service returned an empty user ID",
					nil,
				)
			}

			if response.UserID != request.UserID {
				return nil, apperrors.NewInternal(
					"Users service returned an unexpected user ID",
					nil,
				)
			}

			return &response, nil
		}
	}
}

func (r *RabbitMQ) FetchUser(
	ctx context.Context,
	request FetchUserRequest,
) (*UserResponse, error) {
	if r == nil || r.Connection == nil {
		return nil, apperrors.NewInternal(
			"RabbitMQ connection is not initialized",
			nil,
		)
	}

	if err := ctx.Err(); err != nil {
		return nil, apperrors.NewInternal(
			"Fetch user request was cancelled",
			err,
		)
	}

	if request.UserID == "" {
		return nil, apperrors.NewBadRequest(
			"User ID is required",
			nil,
		)
	}

	if request.Role == "" {
		return nil, apperrors.NewBadRequest(
			"Role is required",
			nil,
		)
	}

	channel, err := r.Connection.Channel()
	if err != nil {
		return nil, apperrors.NewInternal(
			"Failed to open RabbitMQ channel",
			err,
		)
	}
	defer channel.Close()

	queue, err := channel.QueueDeclare(
		"users.fetch",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, apperrors.NewInternal(
			"Failed to declare users queue",
			err,
		)
	}

	replyQueue, err := channel.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		nil,
	)
	if err != nil {
		return nil, apperrors.NewInternal(
			"Failed to declare reply queue",
			err,
		)
	}

	messages, err := channel.Consume(
		replyQueue.Name,
		"",
		true,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, apperrors.NewInternal(
			"Failed to consume reply queue",
			err,
		)
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, apperrors.NewInternal(
			"Failed to encode fetch user request",
			err,
		)
	}

	correlationID := fmt.Sprintf(
		"%s-%d",
		request.UserID,
		time.Now().UnixNano(),
	)

	err = channel.PublishWithContext(
		ctx,
		"",
		queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			ReplyTo:       replyQueue.Name,
			Body:          body,
			DeliveryMode:  amqp.Persistent,
		},
	)
	if err != nil {
		return nil, apperrors.NewInternal(
			"Failed to publish fetch user request",
			err,
		)
	}

	for {
		select {
		case <-ctx.Done():
			return nil, apperrors.NewRequestTimeout(
				"Timed out waiting for user fetch response",
				ctx.Err(),
			)

		case message, ok := <-messages:
			if !ok {
				return nil, apperrors.NewInternal(
					"RabbitMQ reply channel was closed",
					nil,
				)
			}

			if message.CorrelationId != correlationID {
				continue
			}

			var response UserResponse

			if err := json.Unmarshal(message.Body, &response); err != nil {
				return nil, apperrors.NewInternal(
					"Invalid response from users service",
					err,
				)
			}

			if response.Code != "" {
				switch response.Code {
				case "USER_NOT_FOUND":
					return nil, apperrors.NewNotFound(
						response.Message,
						nil,
					)

				default:
					return nil, apperrors.NewInternal(
						response.Message,
						nil,
					)
				}
			}

			if response.UserID == "" {
				return nil, apperrors.NewInternal(
					"Users service returned an empty user ID",
					nil,
				)
			}

			if response.UserID != request.UserID {
				return nil, apperrors.NewInternal(
					"Users service returned an unexpected user ID",
					nil,
				)
			}

			if response.Role != request.Role {
				return nil, apperrors.NewInternal(
					"Users service returned an unexpected user role",
					nil,
				)
			}

			return &response, nil
		}
	}
}
