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
	ID                 uint    `json:"id"`
	SeriesID           string  `json:"serie_id"`
	Description        string  `json:"descricao"`
	CategoryID         uint    `json:"categoria_id"`
	CategoryName       string  `json:"categoria_nome"`
	PaymentSource      string  `json:"fonte_pagamento"`
	Amount             float64 `json:"valor"`
	Month              int     `json:"mes"`
	Year               int     `json:"ano"`
	CurrentInstallment int     `json:"parcela_atual"`
	TotalInstallments  int     `json:"total_parcelas"`
}
