package reports

import "time"

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

type MonthComparisonResponse struct {
	CurrentMonth   int                            `json:"mes_atual"`
	CurrentYear    int                            `json:"ano_atual"`
	ComparedMonth  int                            `json:"mes_comparado"`
	ComparedYear   int                            `json:"ano_comparado"`
	Summary        MonthComparisonSummary         `json:"resumo"`
	Categories     []MonthComparisonCategory      `json:"categorias"`
	PaymentSources []MonthComparisonPaymentSource `json:"fontes_pagamento"`
	ExpenseTypes   []MonthComparisonExpenseType   `json:"tipos_despesa"`
	Insights       []string                       `json:"insights"`
}

type MonthComparisonSummary struct {
	CurrentIncome     float64 `json:"receitas_atual"`
	PreviousIncome    float64 `json:"receitas_comparado"`
	IncomeDifference  float64 `json:"diferenca_receitas"`
	IncomePercentage  float64 `json:"percentual_receitas"`
	IncomeStatus      string  `json:"status_receitas"`
	CurrentExpense    float64 `json:"despesas_atual"`
	PreviousExpense   float64 `json:"despesas_comparado"`
	ExpenseDifference float64 `json:"diferenca_despesas"`
	ExpensePercentage float64 `json:"percentual_despesas"`
	ExpenseStatus     string  `json:"status_despesas"`
	CurrentBalance    float64 `json:"saldo_atual"`
	PreviousBalance   float64 `json:"saldo_comparado"`
	BalanceDifference float64 `json:"diferenca_saldo"`
	BalancePercentage float64 `json:"percentual_saldo"`
	BalanceStatus     string  `json:"status_saldo"`
}

type MonthComparisonCategory struct {
	CategoryID     uint    `json:"categoria_id"`
	CategoryName   string  `json:"categoria_nome"`
	CurrentAmount  float64 `json:"valor_atual"`
	PreviousAmount float64 `json:"valor_comparado"`
	Difference     float64 `json:"diferenca"`
	Percentage     float64 `json:"percentual"`
	Status         string  `json:"status"`
}

type MonthComparisonPaymentSource struct {
	PaymentSource  string  `json:"fonte_pagamento"`
	CurrentAmount  float64 `json:"valor_atual"`
	PreviousAmount float64 `json:"valor_comparado"`
	Difference     float64 `json:"diferenca"`
	Percentage     float64 `json:"percentual"`
	Status         string  `json:"status"`
}

type MonthComparisonExpenseType struct {
	Type           string  `json:"tipo"`
	CurrentAmount  float64 `json:"valor_atual"`
	PreviousAmount float64 `json:"valor_comparado"`
	Difference     float64 `json:"diferenca"`
	Percentage     float64 `json:"percentual"`
	Status         string  `json:"status"`
}

type InstallmentCommitmentsResponse struct {
	BaseMonth int                           `json:"mes_base"`
	BaseYear  int                           `json:"ano_base"`
	Months    int                           `json:"meses"`
	Summary   InstallmentCommitmentsSummary `json:"resumo"`
	Purchases []InstallmentPurchaseSummary  `json:"compras"`
	Timeline  []InstallmentMonthSummary     `json:"linha_do_tempo"`
}

type InstallmentCommitmentsSummary struct {
	OriginalTotal         float64                   `json:"total_original"`
	PaidTotal             float64                   `json:"total_pago"`
	RemainingTotal        float64                   `json:"total_restante"`
	PaidInstallments      int                       `json:"parcelas_pagas"`
	RemainingInstallments int                       `json:"parcelas_restantes"`
	TotalPurchases        int                       `json:"total_compras"`
	HeaviestMonth         *InstallmentHeaviestMonth `json:"mes_mais_pesado"`
}

type InstallmentHeaviestMonth struct {
	Month int     `json:"mes"`
	Year  int     `json:"ano"`
	Total float64 `json:"total"`
}

type InstallmentPurchaseSummary struct {
	SeriesID              string                  `json:"serie_id"`
	Description           string                  `json:"descricao"`
	CategoryID            uint                    `json:"categoria_id"`
	CategoryName          string                  `json:"categoria_nome"`
	PaymentSource         string                  `json:"fonte_pagamento"`
	InstallmentAmount     float64                 `json:"valor_parcela"`
	OriginalTotal         float64                 `json:"total_original"`
	PaidTotal             float64                 `json:"total_pago"`
	RemainingTotal        float64                 `json:"total_restante"`
	PaidInstallments      int                     `json:"parcelas_pagas"`
	RemainingInstallments int                     `json:"parcelas_restantes"`
	TotalInstallments     int                     `json:"total_parcelas"`
	FirstMonth            int                     `json:"primeiro_mes"`
	FirstYear             int                     `json:"primeiro_ano"`
	LastMonth             int                     `json:"ultimo_mes"`
	LastYear              int                     `json:"ultimo_ano"`
	NextInstallment       *InstallmentItemSummary `json:"proxima_parcela"`
}

type InstallmentMonthSummary struct {
	Month        int                      `json:"mes"`
	Year         int                      `json:"ano"`
	Total        float64                  `json:"total"`
	Installments []InstallmentItemSummary `json:"parcelas"`
}

type InstallmentItemSummary struct {
	ID                 uint       `json:"id"`
	SeriesID           string     `json:"serie_id"`
	Description        string     `json:"descricao"`
	CategoryID         uint       `json:"categoria_id"`
	CategoryName       string     `json:"categoria_nome"`
	PaymentSource      string     `json:"fonte_pagamento"`
	Amount             float64    `json:"valor"`
	Month              int        `json:"mes"`
	Year               int        `json:"ano"`
	CurrentInstallment int        `json:"parcela_atual"`
	TotalInstallments  int        `json:"total_parcelas"`
	IsAdvanced         bool       `json:"is_advanced"`
	AdvancedAt         *time.Time `json:"advanced_at,omitempty"`
}
