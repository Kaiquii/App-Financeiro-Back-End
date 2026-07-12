package auth

import "time"

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=3"`
	Code     string `json:"code" binding:"required,len=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=3"`
}

type User struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Name            string     `json:"name"`
	Email           string     `gorm:"uniqueIndex" json:"email"`
	Password        string     `json:"-"`
	Role            string     `gorm:"type:varchar(20);default:user" json:"role"`
	AvatarURL       string     `json:"avatar_url"`
	AccessBlocked   bool       `gorm:"default:false" json:"access_blocked"`
	AccessBlockedAt *time.Time `json:"access_blocked_at,omitempty"`
}

type PasswordResetToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Email     string    `gorm:"index" json:"email"`
	IPAddress string    `gorm:"index" json:"ip_address"`
	CodeHash  string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

type RegistrationCode struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"index" json:"email"`
	CodeHash  string    `json:"-"`
	IPAddress string    `gorm:"index" json:"ip_address"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

type RequestRegisterCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Code        string `json:"code" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=3"`
}

type DeleteUserRequest struct {
	ID uint `json:"id" binding:"required"`
}
