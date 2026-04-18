package expenses

import "time"

type Expense struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	UserID         uint      `json:"user_id"`
	SeriesID       string    `json:"series_id" gorm:"index"`
	CategoryID     uint      `json:"category_id"`
	Amount         float64   `json:"amount"`
	Description    string    `json:"description"`
	PaymentSource  string    `json:"payment_source"`
	Date           time.Time `json:"date"`
	Month          int       `json:"month"`
	Year           int       `json:"year"`
	Type           string    `json:"type"`
	Installments   int       `json:"installments"`
	CurrentInstall int       `json:"current_installment"`
}

type CreateExpenseRequest struct {
	Amount        float64   `json:"amount" binding:"required"`
	Description   string    `json:"description" binding:"required"`
	CategoryID    uint      `json:"category_id"`
	PaymentSource string    `json:"payment_source"`
	Date          time.Time `json:"date" binding:"required"`
	Type          string    `json:"type" binding:"required"`
	Installments  int       `json:"installments"`
}
