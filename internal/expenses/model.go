package expenses

import "time"

type Expense struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	UserID         uint           `json:"user_id"`
	SeriesID       string         `json:"series_id" gorm:"index"`
	CategoryID     uint           `json:"category_id"`
	Amount         float64        `json:"amount"`
	Description    string         `json:"description"`
	Notes          string         `json:"notes" gorm:"type:text"`
	PaymentSource  string         `json:"payment_source"`
	PaymentSplits  []PaymentSplit `json:"payment_splits" gorm:"foreignKey:ExpenseID"`
	Date           time.Time      `json:"date"`
	Month          int            `json:"month"`
	Year           int            `json:"year"`
	Type           string         `json:"type"`
	Installments   int            `json:"installments"`
	CurrentInstall int            `json:"current_installment"`
	IsPaid         bool           `json:"is_paid"`
	PaidAt         *time.Time     `json:"paid_at,omitempty"`
	IsAdvanced     bool           `json:"is_advanced" gorm:"default:false;not null"`
	AdvancedAt     *time.Time     `json:"advanced_at,omitempty" gorm:"index"`
}

type PaymentSplit struct {
	ID            uint    `json:"id" gorm:"primaryKey"`
	ExpenseID     uint    `json:"expense_id" gorm:"index"`
	PaymentSource string  `json:"payment_source"`
	Amount        float64 `json:"amount"`
}

func (PaymentSplit) TableName() string {
	return "expense_payment_splits"
}

type PaymentSplitInput struct {
	PaymentSource string  `json:"payment_source" binding:"required"`
	Amount        float64 `json:"amount" binding:"required"`
}

type CreateExpenseRequest struct {
	Amount        float64             `json:"amount" binding:"required"`
	Description   string              `json:"description" binding:"required"`
	Notes         string              `json:"notes"`
	CategoryID    uint                `json:"category_id"`
	PaymentSource string              `json:"payment_source"`
	PaymentSplits []PaymentSplitInput `json:"payment_splits"`
	Date          time.Time           `json:"date" binding:"required"`
	Type          string              `json:"type" binding:"required"`
	Installments  int                 `json:"installments"`
}

// UpdatePaymentStatusRequest changes only the visual payment status of an
// expense. It intentionally does not change any financial amount or report.
type UpdatePaymentStatusRequest struct {
	IsPaid *bool `json:"is_paid" binding:"required"`
}

// UpdateAdvanceStatusRequest is intentionally independent from the visual
// payment status. Advancing an expense never changes is_paid or paid_at.
type UpdateAdvanceStatusRequest struct {
	IsAdvanced *bool  `json:"is_advanced" binding:"required"`
	AdvancedAt string `json:"advanced_at"`
}
