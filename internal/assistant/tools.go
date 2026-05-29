package assistant

import (
	"App_Financeiro_Back-end/internal/categories"
	"App_Financeiro_Back-end/internal/database"
	"App_Financeiro_Back-end/internal/expenses"
	"App_Financeiro_Back-end/internal/incomes"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

func toolDeclarations() []FunctionDeclaration {
	return []FunctionDeclaration{
		{
			Name:        "get_monthly_summary",
			Description: "Consulta o resumo financeiro de um usuario em um mes e ano especificos: renda total, despesas totais, gastos por fonte de pagamento e saldo restante.",
			Parameters: objectSchema(map[string]any{
				"month": integerSchema("Mes numerico entre 1 e 12."),
				"year":  integerSchema("Ano com quatro digitos, por exemplo 2026."),
			}, []string{"month", "year"}),
		},
		{
			Name:        "get_category_summary",
			Description: "Consulta quanto o usuario gastou por categoria em um mes/ano, ou no ano inteiro se o mes nao for informado.",
			Parameters: objectSchema(map[string]any{
				"month": integerSchema("Mes numerico entre 1 e 12. Opcional."),
				"year":  integerSchema("Ano com quatro digitos, por exemplo 2026."),
			}, []string{"year"}),
		},
		{
			Name:        "list_expenses",
			Description: "Lista despesas do usuario com filtros opcionais por mes, ano, categoria e fonte de pagamento.",
			Parameters: objectSchema(map[string]any{
				"month":          integerSchema("Mes numerico entre 1 e 12. Opcional."),
				"year":           integerSchema("Ano com quatro digitos. Opcional."),
				"category_name":  stringSchema("Nome da categoria para filtrar. Opcional."),
				"payment_source": stringSchema("Fonte usada no pagamento: Salario, Adiantamento ou Renda Extra. Opcional."),
				"limit":          integerSchema("Quantidade maxima de despesas para retornar. Use no maximo 20."),
			}, []string{}),
		},
		{
			Name:        "create_expense",
			Description: "Prepara ou cadastra uma despesa para o usuario. Use confirm=false quando ainda estiver interpretando o pedido. Use confirm=true apenas se a mensagem atual do usuario confirmar claramente um cadastro ja apresentado.",
			Parameters: objectSchema(map[string]any{
				"description":    stringSchema("Descricao curta da despesa, por exemplo Pao ou Uber."),
				"amount":         numberSchema("Valor da despesa em reais."),
				"category_name":  stringSchema("Nome da categoria. Se nao souber, use vazio."),
				"payment_source": enumSchema("Fonte usada no pagamento.", []string{"Salario", "Adiantamento", "Renda Extra"}),
				"month":          integerSchema("Mes numerico entre 1 e 12."),
				"year":           integerSchema("Ano com quatro digitos."),
				"day":            integerSchema("Dia do mes. Se o usuario nao informar, use 1."),
				"type":           enumSchema("Tipo da despesa.", []string{"Unica", "Parcelada", "Fixa"}),
				"installments":   integerSchema("Quantidade de parcelas. Use 1 para despesa unica."),
				"confirm":        boolSchema("Verdadeiro somente quando o usuario confirmar explicitamente o cadastro."),
			}, []string{"description", "amount", "payment_source", "month", "year", "day", "type", "installments", "confirm"}),
		},
	}
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func stringSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func integerSchema(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}

func numberSchema(description string) map[string]any {
	return map[string]any{"type": "number", "description": description}
}

func boolSchema(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}

func enumSchema(description string, values []string) map[string]any {
	return map[string]any{"type": "string", "description": description, "enum": values}
}

func executeTool(userID uint, call FunctionCall) (any, error) {
	switch call.Name {
	case "get_monthly_summary":
		month, err := intArg(call.Args, "month")
		if err != nil {
			return nil, err
		}
		year, err := intArg(call.Args, "year")
		if err != nil {
			return nil, err
		}
		return getMonthlySummary(userID, month, year)
	case "get_category_summary":
		year, err := intArg(call.Args, "year")
		if err != nil {
			return nil, err
		}
		month, _ := intArg(call.Args, "month")
		return getCategorySummary(userID, month, year)
	case "list_expenses":
		return listExpenses(userID, call.Args)
	case "create_expense":
		return createExpenseFromTool(userID, call.Args)
	default:
		return nil, fmt.Errorf("ferramenta nao suportada: %s", call.Name)
	}
}

func getMonthlySummary(userID uint, month int, year int) (map[string]any, error) {
	if err := validateMonthYear(month, year); err != nil {
		return nil, err
	}

	var incomesList []incomes.Income
	if err := database.DB.Where("user_id = ? AND month = ? AND year = ?", userID, month, year).Find(&incomesList).Error; err != nil {
		return nil, err
	}

	var expensesList []expenses.Expense
	if err := database.DB.Where("user_id = ? AND month = ? AND year = ?", userID, month, year).Find(&expensesList).Error; err != nil {
		return nil, err
	}

	var totalIncome, totalExpense float64
	var totalSalary, totalAdiantamento, totalRendaExtra float64
	var totalSpentSalary, totalSpentAdiantamento, totalSpentRendaExtra float64

	for _, income := range incomesList {
		totalIncome += income.Amount
		switch normalizeMoneySource(income.Source) {
		case "salario":
			totalSalary += income.Amount
		case "adiantamento":
			totalAdiantamento += income.Amount
		case "renda_extra":
			totalRendaExtra += income.Amount
		}
	}

	for _, expense := range expensesList {
		totalExpense += expense.Amount
		switch normalizeMoneySource(expense.PaymentSource) {
		case "salario":
			totalSpentSalary += expense.Amount
		case "adiantamento":
			totalSpentAdiantamento += expense.Amount
		case "renda_extra":
			totalSpentRendaExtra += expense.Amount
		}
	}

	return map[string]any{
		"month":                    month,
		"year":                     year,
		"total_income":             roundMoney(totalIncome),
		"total_expense":            roundMoney(totalExpense),
		"total_geral_disponivel":   roundMoney(totalIncome - totalExpense),
		"salario":                  roundMoney(totalSalary),
		"adiantamento":             roundMoney(totalAdiantamento),
		"renda_extra_amt":          roundMoney(totalRendaExtra),
		"total_gasto_salario":      roundMoney(totalSpentSalary),
		"total_gasto_adiantamento": roundMoney(totalSpentAdiantamento),
		"total_gasto_renda_extra":  roundMoney(totalSpentRendaExtra),
		"restante_salario":         roundMoney(totalSalary - totalSpentSalary),
		"restante_adiantamento":    roundMoney(totalAdiantamento - totalSpentAdiantamento),
		"restante_renda_extra":     roundMoney(totalRendaExtra - totalSpentRendaExtra),
	}, nil
}

func getCategorySummary(userID uint, month int, year int) ([]map[string]any, error) {
	if year < 2000 {
		return nil, errors.New("ano invalido")
	}
	if month != 0 {
		if err := validateMonthYear(month, year); err != nil {
			return nil, err
		}
	}

	type row struct {
		CategoryID   uint
		CategoryName string
		TotalAmount  float64
	}

	var rows []row
	query := database.DB.Table("expenses").
		Select("expenses.category_id, categories.name as category_name, sum(expenses.amount) as total_amount").
		Joins("left join categories on categories.id = expenses.category_id").
		Where("expenses.user_id = ? AND expenses.year = ?", userID, year)

	if month != 0 {
		query = query.Where("expenses.month = ?", month)
	}

	if err := query.Group("expenses.category_id, categories.name").Scan(&rows).Error; err != nil {
		return nil, err
	}

	var total float64
	for _, row := range rows {
		total += row.TotalAmount
	}

	results := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		percentage := 0.0
		if total > 0 {
			percentage = (row.TotalAmount / total) * 100
		}
		results = append(results, map[string]any{
			"category_id":   row.CategoryID,
			"category_name": row.CategoryName,
			"total_amount":  roundMoney(row.TotalAmount),
			"percentage":    roundMoney(percentage),
		})
	}

	return results, nil
}

func listExpenses(userID uint, args map[string]any) ([]expenses.Expense, error) {
	query := database.DB.Where("user_id = ?", userID)

	if month, err := intArg(args, "month"); err == nil && month > 0 {
		query = query.Where("month = ?", month)
	}
	if year, err := intArg(args, "year"); err == nil && year > 0 {
		query = query.Where("year = ?", year)
	}
	if paymentSource, _ := stringArg(args, "payment_source"); paymentSource != "" {
		query = query.Where("LOWER(payment_source) IN ?", paymentSourceVariants(paymentSource))
	}
	if categoryName, _ := stringArg(args, "category_name"); categoryName != "" {
		var category categories.Category
		if err := database.DB.Where("user_id = ? AND LOWER(name) = ?", userID, strings.ToLower(categoryName)).First(&category).Error; err != nil {
			return []expenses.Expense{}, nil
		}
		query = query.Where("category_id = ?", category.ID)
	}

	limit, _ := intArg(args, "limit")
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	var expensesList []expenses.Expense
	if err := query.Order("date desc").Limit(limit).Find(&expensesList).Error; err != nil {
		return nil, err
	}

	return expensesList, nil
}

func createExpenseFromTool(userID uint, args map[string]any) (map[string]any, error) {
	description, err := stringArg(args, "description")
	if err != nil {
		return nil, err
	}
	amount, err := floatArg(args, "amount")
	if err != nil {
		return nil, err
	}
	paymentSource, err := stringArg(args, "payment_source")
	if err != nil {
		return nil, err
	}
	month, err := intArg(args, "month")
	if err != nil {
		return nil, err
	}
	year, err := intArg(args, "year")
	if err != nil {
		return nil, err
	}
	day, err := intArg(args, "day")
	if err != nil || day <= 0 {
		day = 1
	}
	expenseType, err := stringArg(args, "type")
	if err != nil {
		expenseType = "Unica"
	}
	installments, _ := intArg(args, "installments")
	if installments <= 0 {
		installments = 1
	}
	confirm, _ := boolArg(args, "confirm")

	if err := validateMonthYear(month, year); err != nil {
		return nil, err
	}

	date := time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC)
	if date.Month() != time.Month(month) {
		return nil, errors.New("dia invalido para o mes informado")
	}

	categoryID := uint(0)
	categoryName, _ := stringArg(args, "category_name")
	if categoryName != "" {
		category, err := findOrCreateCategory(userID, categoryName, confirm)
		if err != nil {
			return nil, err
		}
		categoryID = category.ID
		categoryName = category.Name
	}

	preview := map[string]any{
		"status":         "needs_confirmation",
		"description":    description,
		"amount":         roundMoney(amount),
		"category_id":    categoryID,
		"category_name":  categoryName,
		"payment_source": normalizePaymentSource(paymentSource),
		"date":           date.Format("2006-01-02"),
		"day":            day,
		"month":          month,
		"year":           year,
		"type":           normalizeExpenseTypeForCreate(expenseType),
		"installments":   installments,
		"confirm":        false,
	}

	if !confirm {
		return preview, nil
	}

	expenseType = normalizeExpenseTypeForCreate(expenseType)
	loopCount := 1
	seriesID := ""

	switch expenseType {
	case "Unica":
		installments = 1
	case "Parcelada":
		loopCount = installments
		seriesID = fmt.Sprintf("expense-%d-%d", userID, time.Now().UnixNano())
	case "Fixa":
		loopCount = 120
		installments = 0
		seriesID = fmt.Sprintf("expense-%d-%d", userID, time.Now().UnixNano())
	default:
		return nil, errors.New("tipo de despesa invalido")
	}

	for i := 1; i <= loopCount; i++ {
		installmentDate := date.AddDate(0, i-1, 0)
		newExpense := expenses.Expense{
			UserID:         userID,
			SeriesID:       seriesID,
			Amount:         amount,
			Description:    description,
			CategoryID:     categoryID,
			PaymentSource:  normalizePaymentSource(paymentSource),
			Date:           installmentDate,
			Month:          int(installmentDate.Month()),
			Year:           installmentDate.Year(),
			Type:           expenseType,
			Installments:   installments,
			CurrentInstall: i,
		}
		if err := database.DB.Create(&newExpense).Error; err != nil {
			return nil, err
		}
	}

	preview["status"] = "created"
	preview["created_count"] = loopCount
	preview["confirm"] = true
	return preview, nil
}

func validateMonthYear(month int, year int) error {
	if month < 1 || month > 12 {
		return errors.New("mes invalido")
	}
	if year < 2000 {
		return errors.New("ano invalido")
	}
	return nil
}

func normalizeMoneySource(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "salario", "salário":
		return "salario"
	case "adiantamento":
		return "adiantamento"
	case "renda extra", "renda_extra":
		return "renda_extra"
	default:
		return ""
	}
}

func normalizePaymentSource(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "salario", "salário":
		return "Salario"
	case "adiantamento":
		return "Adiantamento"
	case "renda extra", "renda_extra":
		return "Renda Extra"
	default:
		return source
	}
}

func paymentSourceVariants(source string) []string {
	normalized := normalizeMoneySource(source)
	switch normalized {
	case "salario":
		return []string{"salario", "salário", "salã¡rio"}
	case "adiantamento":
		return []string{"adiantamento"}
	case "renda_extra":
		return []string{"renda extra", "renda_extra"}
	default:
		return []string{strings.ToLower(strings.TrimSpace(source))}
	}
}

func normalizeExpenseTypeForCreate(expenseType string) string {
	switch strings.TrimSpace(strings.ToLower(expenseType)) {
	case "unica", "única":
		return "Unica"
	case "parcelada":
		return "Parcelada"
	case "fixa":
		return "Fixa"
	default:
		return "Unica"
	}
}

func findOrCreateCategory(userID uint, categoryName string, shouldCreate bool) (categories.Category, error) {
	categoryName = strings.TrimSpace(categoryName)
	var category categories.Category
	if categoryName == "" {
		return category, nil
	}

	err := database.DB.Where("user_id = ? AND LOWER(name) = ?", userID, strings.ToLower(categoryName)).First(&category).Error
	if err == nil {
		return category, nil
	}

	if !shouldCreate {
		return categories.Category{
			UserID: userID,
			Name:   categoryName,
		}, nil
	}

	category = categories.Category{
		UserID: userID,
		Name:   categoryName,
	}
	if err := database.DB.Create(&category).Error; err != nil {
		return category, err
	}

	return category, nil
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func stringArg(args map[string]any, key string) (string, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return "", fmt.Errorf("argumento obrigatorio ausente: %s", key)
	}
	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("argumento %s deve ser texto", key)
	}
	return strings.TrimSpace(str), nil
}

func intArg(args map[string]any, key string) (int, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return 0, fmt.Errorf("argumento obrigatorio ausente: %s", key)
	}
	switch typed := value.(type) {
	case float64:
		return int(typed), nil
	case int:
		return typed, nil
	default:
		return 0, fmt.Errorf("argumento %s deve ser inteiro", key)
	}
}

func floatArg(args map[string]any, key string) (float64, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return 0, fmt.Errorf("argumento obrigatorio ausente: %s", key)
	}
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case int:
		return float64(typed), nil
	default:
		return 0, fmt.Errorf("argumento %s deve ser numero", key)
	}
}

func boolArg(args map[string]any, key string) (bool, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return false, fmt.Errorf("argumento obrigatorio ausente: %s", key)
	}
	boolValue, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("argumento %s deve ser booleano", key)
	}
	return boolValue, nil
}
