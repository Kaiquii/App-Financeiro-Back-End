package expenses

import "time"

type Expense struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	UserID         uint      `json:"user_id"`
	Amount         float64   `json:"amount"`
	Description    string    `json:"description"`
	Category       string    `json:"category"`
	PaymentSource  string    `json:"payment_source"`
	Date           time.Time `json:"date"`
	Month          int       `json:"month"`
	Year           int       `json:"year"`
	Type           string    `json:"type"`
	Installments   int       `json:"installments"`
	CurrentInstall int       `json:"current_installment"`
}
