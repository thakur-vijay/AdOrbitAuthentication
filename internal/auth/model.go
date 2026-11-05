package auth

import "time"

type LoginRequestDTO struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type Credential struct {
	UserID          string    `json:"user_id"`
	Email           string    `json:"email"`
	PasswordHash    string    `json:"password_hash"`
	Status          string    `json:"status"`
	IsEmailVerified bool      `json:"is_email_verified"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SignupRequestDTO struct {
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email" validate:"required"`
	Role     string `json:"role" validate:"required,oneof=advertiser publisher"`
	Password string `json:"password" validate:"required,valid_password"`
}

type ResendOTPDTO struct {
	Email string `json:"email" validate:"required,email"`
}

type LoginResponseDTO struct {
	User         LoginUserDTO `json:"user"`
	AccessToken  *string      `json:"access_token"`
	RefreshToken *string      `json:"refresh_token"`
}

type LoginUserDTO struct {
	ID              string    `json:"id"`
	FullName        string    `json:"full_name"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	IsEmailVerified bool      `json:"is_email_verified"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type RefreshRequestDTO struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshResponseDTO struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
