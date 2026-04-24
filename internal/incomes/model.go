package incomes

type Income struct {
	ID     uint    `json:"id" gorm:"primaryKey"`
	UserID uint    `json:"user_id"`
	Source string  `json:"source"`
	Amount float64 `json:"amount"`
	Month  int     `json:"month"`
	Year   int     `json:"year"`
}

type CreateIncomeRequest struct {
	Source       string  `json:"source" binding:"required"`
	Amount       float64 `json:"amount" binding:"required"`
	Month        int     `json:"month" binding:"required"`
	Year         int     `json:"year" binding:"required"`
	Type         string  `json:"type"`
	RepeatFuture bool    `json:"repeat_future"`
}
