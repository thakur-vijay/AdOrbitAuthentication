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

type CreateUserResponse struct {
	UserID  string `json:"user_id,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (r *RabbitMQ) CreateUser(
	ctx context.Context,
	request CreateUserRequest,
) (string, error) {
	if r == nil || r.Connection == nil {
		return "", apperrors.NewInternal(
			"RabbitMQ connection is not initialized",
			nil,
		)
	}

	if err := ctx.Err(); err != nil {
		return "", apperrors.NewInternal(
			"Create user request was cancelled",
			err,
		)
	}

	if request.UserID == "" {
		return "", apperrors.NewBadRequest(
			"User ID is required",
			nil,
		)
	}

	channel, err := r.Connection.Channel()
	if err != nil {
		return "", apperrors.NewInternal(
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
		return "", apperrors.NewInternal(
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
		return "", apperrors.NewInternal(
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
		return "", apperrors.NewInternal(
			"Failed to consume reply queue",
			err,
		)
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", apperrors.NewInternal(
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
		return "", apperrors.NewInternal(
			"Failed to publish create user request",
			err,
		)
	}

	for {
		select {
		case <-ctx.Done():
			return "", apperrors.NewRequestTimeout(
				"Timed out waiting for user creation response",
				ctx.Err(),
			)

		case message, ok := <-messages:
			if !ok {
				return "", apperrors.NewInternal(
					"RabbitMQ reply channel was closed",
					nil,
				)
			}

			if message.CorrelationId != correlationID {
				continue
			}

			var response CreateUserResponse

			if err := json.Unmarshal(message.Body, &response); err != nil {
				return "", apperrors.NewInternal(
					"Invalid response from users service",
					err,
				)
			}

			if response.Code != "" {
				switch response.Code {
				case "USER_ALREADY_EXISTS":
					return "", apperrors.NewConflict(
						response.Message,
						nil,
					)

				default:
					return "", apperrors.NewInternal(
						response.Message,
						nil,
					)
				}
			}

			if response.UserID == "" {
				return "", apperrors.NewInternal(
					"Users service returned an empty user ID",
					nil,
				)
			}

			if response.UserID != request.UserID {
				return "", apperrors.NewInternal(
					"Users service returned an unexpected user ID",
					nil,
				)
			}

			return response.UserID, nil
		}
	}
}
