package incomes

type Income struct {
	ID     uint    `json:"id" gorm:"primaryKey"`
	UserID uint    `json:"user_id"`
	Source string  `json:"source"`
	Amount float64 `json:"amount"`
	Month  int     `json:"month"`
	Year   int     `json:"year"`
}
