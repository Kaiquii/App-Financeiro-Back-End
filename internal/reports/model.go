package reports

type ChartResult struct {
	Month   int     `json:"month"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
}

type CategoryResult struct {
	CategoryID   uint    `json:"category_id"`
	CategoryName string  `json:"category_name"`
	TotalAmount  float64 `json:"total_amount"`
	Percentage   float64 `json:"percentage"`
}
