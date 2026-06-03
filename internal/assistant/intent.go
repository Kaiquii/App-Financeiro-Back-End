package assistant

import (
	"App_Financeiro_Back-end/internal/categories"
	"App_Financeiro_Back-end/internal/database"
	"App_Financeiro_Back-end/internal/expenses"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

type financialIntent struct {
	Intent        string  `json:"intent"`
	Reply         string  `json:"reply"`
	Month         int     `json:"month"`
	Year          int     `json:"year"`
	CategoryName  string  `json:"category_name"`
	PaymentSource string  `json:"payment_source"`
	Description   string  `json:"description"`
	Amount        float64 `json:"amount"`
	Day           int     `json:"day"`
	Type          string  `json:"type"`
	Installments  int     `json:"installments"`
}

func handleFinancialMessageWithContext(client textGenerator, userID uint, message string) (bool, string, string, any) {
	intent, err := interpretFinancialIntent(client, userID, message)
	if err != nil {
		if retryAfter, ok := geminiQuotaInfo(err); ok {
			log.Printf("Assistant intent blocked by Gemini quota retry_after=%ds", retryAfter)
			return true, quotaReply(retryAfter), "", map[string]any{
				"error_code":          "gemini_quota_exceeded",
				"retry_after_seconds": retryAfter,
			}
		}
		log.Printf("Assistant intent failed, skipping second Gemini call: %v", err)
		return true, "Nao consegui interpretar sua mensagem agora. Tente novamente em alguns instantes.", "", map[string]any{
			"error_code": "assistant_intent_failed",
		}
	}

	log.Printf(
		"Assistant intent result intent=%s month=%d year=%d category=%q source=%q has_amount=%t",
		intent.Intent,
		intent.Month,
		intent.Year,
		intent.CategoryName,
		intent.PaymentSource,
		intent.Amount > 0,
	)

	switch intent.Intent {
	case "general", "app_help":
		reply := strings.TrimSpace(intent.Reply)
		if reply == "" {
			reply = "Claro. Como posso te ajudar com suas financas?"
		}
		return true, reply, intent.Intent, nil

	case "create_expense":
		args := map[string]any{
			"description":    intent.Description,
			"amount":         intent.Amount,
			"category_name":  intent.CategoryName,
			"payment_source": intent.PaymentSource,
			"month":          float64(intent.Month),
			"year":           float64(intent.Year),
			"day":            float64(intent.Day),
			"type":           intent.Type,
			"installments":   float64(intent.Installments),
			"confirm":        false,
		}

		result, err := createExpenseFromTool(userID, args)
		if err != nil {
			return false, "", "", nil
		}

		reply := localToolReply("create_expense", result)
		return true, reply, "create_expense", result

	case "category_expenses":
		if strings.TrimSpace(intent.CategoryName) == "" {
			result, err := getAllCategoryExpenseSummary(userID, intent.Month, intent.Year, intent.PaymentSource)
			if err != nil {
				return false, "", "", nil
			}

			reply := localToolReply("get_all_category_expenses", result)
			return true, reply, "get_all_category_expenses", result
		}

		result, err := getCategoryExpenseSummary(userID, intent.Month, intent.Year, intent.CategoryName, intent.PaymentSource)
		if err != nil {
			return false, "", "", nil
		}

		reply := localToolReply("get_category_expenses", result)
		return true, reply, "get_category_expenses", result

	case "list_categories":
		result, err := listUserCategories(userID)
		if err != nil {
			return false, "", "", nil
		}

		reply := localToolReply("list_categories", result)
		return true, reply, "list_categories", result

	case "all_category_expenses":
		result, err := getAllCategoryExpenseSummary(userID, intent.Month, intent.Year, intent.PaymentSource)
		if err != nil {
			return false, "", "", nil
		}

		reply := localToolReply("get_all_category_expenses", result)
		return true, reply, "get_all_category_expenses", result

	case "monthly_summary":
		result, err := getMonthlySummary(userID, intent.Month, intent.Year)
		if err != nil {
			return false, "", "", nil
		}

		reply := localToolReply("get_monthly_summary", result)
		return true, reply, "get_monthly_summary", result

	default:
		return false, "", "", nil
	}
}

func interpretFinancialIntent(client textGenerator, userID uint, message string) (financialIntent, error) {
	now := time.Now()
	knownCategories := categoryNamesForPrompt(userID)
	prompt := "Interprete a mensagem do usuario para o App Financeiro e responda somente JSON valido, sem markdown.\n" +
		"Intencoes possiveis:\n" +
		"- general: cumprimento, conversa comum ou pergunta que nao exige dados financeiros.\n" +
		"- app_help: pergunta sobre como usar o app.\n" +
		"- monthly_summary: pergunta sobre total gasto, saldo, salario, renda ou resumo de um periodo.\n" +
		"- category_expenses: pergunta sobre gastos de uma categoria especifica. Se o usuario disser categoria seguida de um nome, use esta intencao.\n" +
		"- list_categories: pergunta sobre quais categorias o usuario tem cadastradas.\n" +
		"- all_category_expenses: pergunta sobre gastos em cada categoria, divisao por categoria, ranking de categorias ou todas as categorias, sem escolher uma categoria especifica.\n" +
		"- create_expense: pedido para cadastrar/adicionar/registrar uma despesa.\n\n" +
		"Campos do JSON: intent, reply, month, year, category_name, payment_source, description, amount, day, type, installments.\n" +
		"Para general e app_help, preencha reply com uma resposta curta e util para o usuario.\n" +
		"Use month/year do periodo citado. Se o usuario disser 'este mes', 'esse mes' ou 'ate agora', use " + strconv.Itoa(int(now.Month())) + "/" + strconv.Itoa(now.Year()) + ".\n" +
		"Categorias cadastradas do usuario: " + knownCategories + ".\n" +
		"Se o usuario citar uma categoria, preencha category_name com o nome exato entendido ou com o nome cadastrado correspondente. Se nao houver categoria especifica, use string vazia.\n" +
		"Se category_name estiver preenchido, a intent deve ser category_expenses quando a pergunta for sobre gastos ou valores.\n" +
		"payment_source deve ser Salario, Adiantamento, Renda Extra ou string vazia.\n" +
		"type deve ser Unica, Parcelada, Fixa ou Unica se nao informado.\n" +
		"Para create_expense, extraia description, amount, category_name, payment_source, month, year, day, type e installments.\n" +
		"Para intents que nao precisam de algum campo, use zero ou string vazia.\n" +
		"Mensagem: " + message

	text, err := client.generateText(GeminiRequest{
		SystemInstruction: &GeminiContent{Parts: []GeminiPart{{Text: "Voce interpreta intencoes financeiras e responde apenas JSON valido."}}},
		Contents: []GeminiContent{{
			Role:  "user",
			Parts: []GeminiPart{{Text: prompt}},
		}},
		GenerationConfig: map[string]any{"temperature": 0},
		DebugStage:       "assistant_intent",
	})
	if err != nil {
		return financialIntent{}, err
	}

	text = cleanJSONText(text)
	intent, err := parseFinancialIntent(text)
	if err != nil {
		log.Printf("Assistant intent JSON parse failed raw=%s error=%v", compactLogText(text, 300), err)
		return financialIntent{}, err
	}

	if intent.Month == 0 {
		intent.Month = int(now.Month())
	}
	if intent.Year == 0 {
		intent.Year = now.Year()
	}
	if intent.Day == 0 {
		intent.Day = 1
	}
	if intent.Type == "" {
		intent.Type = "Unica"
	}
	if intent.Installments == 0 {
		intent.Installments = 1
	}

	return normalizeInterpretedIntent(intent), nil
}

func categoryNamesForPrompt(userID uint) string {
	var categoriesList []categories.Category
	if err := database.DB.Where("user_id = ?", userID).Order("name asc").Find(&categoriesList).Error; err != nil {
		return "nenhuma"
	}

	names := make([]string, 0, len(categoriesList))
	for _, category := range categoriesList {
		name := strings.TrimSpace(category.Name)
		if name != "" {
			names = append(names, name)
		}
	}

	if len(names) == 0 {
		return "nenhuma"
	}
	return strings.Join(names, ", ")
}

func normalizeInterpretedIntent(intent financialIntent) financialIntent {
	intent.Intent = strings.TrimSpace(intent.Intent)
	if strings.TrimSpace(intent.CategoryName) == "" {
		return intent
	}

	switch intent.Intent {
	case "monthly_summary", "all_category_expenses":
		intent.Intent = "category_expenses"
	}

	return intent
}

func cleanJSONText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return strings.TrimSpace(text[start : end+1])
	}

	return text
}

func parseFinancialIntent(text string) (financialIntent, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return financialIntent{}, err
	}

	return financialIntent{
		Intent:        stringValue(raw["intent"]),
		Reply:         stringValue(raw["reply"]),
		Month:         intValue(raw["month"]),
		Year:          intValue(raw["year"]),
		CategoryName:  stringValue(raw["category_name"]),
		PaymentSource: stringValue(raw["payment_source"]),
		Description:   stringValue(raw["description"]),
		Amount:        floatValue(raw["amount"]),
		Day:           intValue(raw["day"]),
		Type:          stringValue(raw["type"]),
		Installments:  intValue(raw["installments"]),
	}, nil
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func intValue(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case int:
		return typed
	case string:
		typed = strings.TrimSpace(typed)
		if typed == "" {
			return 0
		}
		parsed, err := strconv.Atoi(typed)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func floatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case string:
		typed = strings.TrimSpace(strings.ReplaceAll(typed, ",", "."))
		if typed == "" {
			return 0
		}
		parsed, err := strconv.ParseFloat(typed, 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func getCategoryExpenseSummary(userID uint, month int, year int, categoryName string, paymentSource string) (map[string]any, error) {
	var category categories.Category
	err := database.DB.
		Where("user_id = ? AND LOWER(name) = ?", userID, strings.ToLower(strings.TrimSpace(categoryName))).
		First(&category).Error
	if err != nil {
		return map[string]any{
			"month":          month,
			"year":           year,
			"category_name":  categoryName,
			"payment_source": paymentSource,
			"total_expense":  0,
			"monthly_total":  0,
			"percentage":     0,
			"expenses":       []expenses.Expense{},
		}, nil
	}

	query := database.DB.Where("user_id = ? AND month = ? AND year = ? AND category_id = ?", userID, month, year, category.ID)
	if strings.TrimSpace(paymentSource) != "" {
		query = query.Where("LOWER(payment_source) IN ?", paymentSourceVariants(paymentSource))
	}

	var expensesList []expenses.Expense
	if err := query.Order("date desc").Find(&expensesList).Error; err != nil {
		return nil, err
	}

	total := 0.0
	for _, expense := range expensesList {
		total += expense.Amount
	}

	monthlyTotal, err := getMonthlyExpenseTotal(userID, month, year, paymentSource)
	if err != nil {
		return nil, err
	}

	percentage := 0.0
	if monthlyTotal > 0 {
		percentage = (total / monthlyTotal) * 100
	}

	return map[string]any{
		"month":          month,
		"year":           year,
		"category_id":    category.ID,
		"category_name":  category.Name,
		"payment_source": paymentSource,
		"total_expense":  roundMoney(total),
		"monthly_total":  roundMoney(monthlyTotal),
		"percentage":     roundMoney(percentage),
		"expenses":       expensesList,
	}, nil
}

func getMonthlyExpenseTotal(userID uint, month int, year int, paymentSource string) (float64, error) {
	query := database.DB.Table("expenses").
		Where("user_id = ? AND month = ? AND year = ?", userID, month, year)

	if strings.TrimSpace(paymentSource) != "" {
		query = query.Where("LOWER(payment_source) IN ?", paymentSourceVariants(paymentSource))
	}

	var total float64
	err := query.Select("COALESCE(sum(amount), 0)").Scan(&total).Error
	return total, err
}

func listUserCategories(userID uint) (map[string]any, error) {
	var categoriesList []categories.Category
	if err := database.DB.Where("user_id = ?", userID).Order("name asc").Find(&categoriesList).Error; err != nil {
		return nil, err
	}

	names := make([]string, 0, len(categoriesList))
	for _, category := range categoriesList {
		names = append(names, category.Name)
	}

	return map[string]any{
		"total":      len(categoriesList),
		"categories": categoriesList,
		"names":      names,
	}, nil
}

func getAllCategoryExpenseSummary(userID uint, month int, year int, paymentSource string) (map[string]any, error) {
	type row struct {
		CategoryID   uint
		CategoryName string
		TotalAmount  float64
	}

	query := database.DB.Table("expenses").
		Select("expenses.category_id, categories.name as category_name, sum(expenses.amount) as total_amount").
		Joins("left join categories on categories.id = expenses.category_id").
		Where("expenses.user_id = ? AND expenses.month = ? AND expenses.year = ?", userID, month, year)

	if strings.TrimSpace(paymentSource) != "" {
		query = query.Where("LOWER(expenses.payment_source) IN ?", paymentSourceVariants(paymentSource))
	}

	var rows []row
	if err := query.Group("expenses.category_id, categories.name").Order("total_amount desc").Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0, len(rows))
	total := 0.0
	for _, row := range rows {
		total += row.TotalAmount
		items = append(items, map[string]any{
			"category_id":   row.CategoryID,
			"category_name": row.CategoryName,
			"total_amount":  roundMoney(row.TotalAmount),
		})
	}

	return map[string]any{
		"month":          month,
		"year":           year,
		"payment_source": paymentSource,
		"total_expense":  roundMoney(total),
		"categories":     items,
	}, nil
}
