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
	Amount       float64 `json:"amount" binding:"required,gt=0"`
	Month        int     `json:"month" binding:"required,gte=1,lte=12"`
	Year         int     `json:"year" binding:"required,gte=2000"`
	Type         string  `json:"type"`
	RepeatFuture bool    `json:"repeat_future"`
}
