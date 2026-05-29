package assistant

import (
	"App_Financeiro_Back-end/internal/database"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	assistantGroup := rg.Group("/assistant")
	{
		assistantGroup.POST("/chat", chat)
		assistantGroup.GET("/conversations", listConversations)
		assistantGroup.GET("/conversations/:id/messages", listMessages)
		assistantGroup.DELETE("/conversations/:id", deleteConversation)
	}
}

func chat(c *gin.Context) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario nao identificado"})
		return
	}
	userID := userIDObj.(uint)

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados invalidos: " + err.Error()})
		return
	}

	conversation, err := getOrCreateConversation(userID, req.ConversationID, req.Message)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversa nao encontrada"})
		return
	}

	if pending, ok := lastPendingCreateExpense(userID, conversation.ID); ok && isConfirmationMessage(req.Message) {
		if err := saveMessage(userID, conversation.ID, "user", req.Message, "", nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar mensagem"})
			return
		}

		pending["confirm"] = true
		toolResult, err := createExpenseFromTool(userID, pending)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "tool_call": "create_expense"})
			return
		}

		reply := localToolReply("create_expense", toolResult)
		if err := saveMessage(userID, conversation.ID, "assistant", reply, "create_expense", toolResult); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar resposta"})
			return
		}

		c.JSON(http.StatusOK, ChatResponse{
			ConversationID: conversation.ID,
			Reply:          reply,
			ToolCall:       "create_expense",
			ToolResult:     toolResult,
		})
		return
	}

	client, err := newAssistantGenerator()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	recentMessages, err := loadRecentMessages(userID, conversation.ID, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar historico"})
		return
	}

	handled, reply, toolCall, toolResult := handleFinancialMessageWithContext(client, userID, req.Message)
	if handled {
		errorCode, retryAfter := responseErrorFields(toolResult)
		if errorCode != "" {
			log.Printf("Assistente retornou erro controlado: %s retry_after=%d", errorCode, retryAfter)
		}
		if err := saveMessage(userID, conversation.ID, "user", req.Message, "", nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar mensagem"})
			return
		}
		if err := saveMessage(userID, conversation.ID, "assistant", reply, toolCall, toolResult); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar resposta"})
			return
		}

		c.JSON(http.StatusOK, ChatResponse{
			ConversationID:    conversation.ID,
			Reply:             reply,
			ToolCall:          toolCall,
			ToolResult:        toolResult,
			ErrorCode:         errorCode,
			RetryAfterSeconds: retryAfter,
		})
		return
	}

	contents := buildConversationFromMessages(recentMessages, req.Message)
	request := GeminiRequest{
		SystemInstruction: &GeminiContent{Parts: []GeminiPart{{Text: systemPrompt()}}},
		Contents:          contents,
		GenerationConfig:  map[string]any{"temperature": 0.2},
		DebugStage:        "assistant_chat",
	}

	reply, err = client.generateText(request)
	if err != nil {
		log.Printf("Erro ao chamar Gemini no assistente: %v", err)
		reply, errorCode, retryAfter := assistantErrorResponse(err)
		_ = saveMessage(userID, conversation.ID, "user", req.Message, "", nil)
		_ = saveMessage(userID, conversation.ID, "assistant", reply, "", nil)
		c.JSON(http.StatusOK, ChatResponse{
			ConversationID:    conversation.ID,
			Reply:             reply,
			ErrorCode:         errorCode,
			RetryAfterSeconds: retryAfter,
		})
		return
	}

	if err := saveMessage(userID, conversation.ID, "user", req.Message, "", nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar mensagem"})
		return
	}

	if reply == "" {
		reply = "Nao consegui interpretar sua mensagem agora. Pode tentar perguntar de outro jeito?"
	}

	if err := saveMessage(userID, conversation.ID, "assistant", reply, "", nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar resposta"})
		return
	}

	c.JSON(http.StatusOK, ChatResponse{ConversationID: conversation.ID, Reply: reply})
}

func listConversations(c *gin.Context) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario nao identificado"})
		return
	}
	userID := userIDObj.(uint)

	var conversations []Conversation
	if err := database.DB.
		Where("user_id = ?", userID).
		Order("updated_at desc").
		Find(&conversations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar conversas"})
		return
	}

	response := make([]ConversationResponse, 0, len(conversations))
	for _, conversation := range conversations {
		response = append(response, conversationResponse(conversation))
	}

	c.JSON(http.StatusOK, gin.H{"total": len(response), "conversations": response})
}

func listMessages(c *gin.Context) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario nao identificado"})
		return
	}
	userID := userIDObj.(uint)

	conversationID := c.Param("id")
	var conversation Conversation
	if err := database.DB.Where("id = ? AND user_id = ?", conversationID, userID).First(&conversation).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversa nao encontrada"})
		return
	}

	var messages []Message
	if err := database.DB.
		Where("conversation_id = ? AND user_id = ?", conversation.ID, userID).
		Order("created_at asc").
		Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar mensagens"})
		return
	}

	response := make([]MessageResponse, 0, len(messages))
	for _, message := range messages {
		response = append(response, messageResponse(message))
	}

	c.JSON(http.StatusOK, gin.H{"conversation": conversationResponse(conversation), "messages": response})
}

func deleteConversation(c *gin.Context) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario nao identificado"})
		return
	}
	userID := userIDObj.(uint)

	conversationID := c.Param("id")
	tx := database.DB.Begin()
	if err := tx.Where("conversation_id = ? AND user_id = ?", conversationID, userID).Delete(&Message{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao apagar mensagens"})
		return
	}
	if err := tx.Where("id = ? AND user_id = ?", conversationID, userID).Delete(&Conversation{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao apagar conversa"})
		return
	}
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Conversa apagada com sucesso"})
}

func conversationResponse(conversation Conversation) ConversationResponse {
	return ConversationResponse{
		ConversationID: conversation.ID,
		Title:          conversation.Title,
		CreatedAt:      conversation.CreatedAt,
		UpdatedAt:      conversation.UpdatedAt,
		DisplayDate:    conversation.CreatedAt.Format("02/01/2006"),
		DisplayTime:    conversation.CreatedAt.Format("15:04"),
		DisplayLabel:   conversation.CreatedAt.Format("02/01/2006 15:04"),
	}
}

func messageResponse(message Message) MessageResponse {
	return MessageResponse{
		MessageID:      message.ID,
		ConversationID: message.ConversationID,
		Role:           message.Role,
		Content:        message.Content,
		ToolCall:       message.ToolCall,
		ToolResult:     message.ToolResult,
		CreatedAt:      message.CreatedAt,
		DisplayTime:    message.CreatedAt.Format("15:04"),
	}
}

func buildConversation(req ChatRequest) []GeminiContent {
	contents := make([]GeminiContent, 0, len(req.History)+1)

	for _, msg := range req.History {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}

		role := "user"
		if strings.EqualFold(msg.Role, "assistant") || strings.EqualFold(msg.Role, "model") {
			role = "model"
		}

		contents = append(contents, GeminiContent{
			Role:  role,
			Parts: []GeminiPart{{Text: content}},
		})
	}

	contents = append(contents, GeminiContent{
		Role:  "user",
		Parts: []GeminiPart{{Text: strings.TrimSpace(req.Message)}},
	})

	return contents
}

func summarizeToolResult(client textGenerator, userMessage string, toolName string, toolResult any) (string, error) {
	resultJSON, err := json.Marshal(toolResult)
	if err != nil {
		return "", err
	}

	prompt := "Mensagem do usuario: " + userMessage + "\n" +
		"Ferramenta executada: " + toolName + "\n" +
		"Resultado JSON: " + string(resultJSON) + "\n\n" +
		"Responda em portugues do Brasil, de forma curta e util. " +
		"Se o status for needs_confirmation, peca confirmacao antes de cadastrar. " +
		"Se o status for created, confirme o cadastro. " +
		"Nao invente valores fora do JSON."

	reply, err := client.generateText(GeminiRequest{
		SystemInstruction: &GeminiContent{Parts: []GeminiPart{{Text: "Voce transforma resultados JSON do App Financeiro em respostas naturais, precisas e curtas."}}},
		Contents: []GeminiContent{{
			Role:  "user",
			Parts: []GeminiPart{{Text: prompt}},
		}},
		GenerationConfig: map[string]any{"temperature": 0.2},
		DebugStage:       "assistant_summarize_tool",
	})
	if err != nil {
		return "", err
	}

	if reply == "" {
		return "Pronto, consultei os dados para voce.", nil
	}
	return reply, nil
}

func isConfirmationMessage(message string) bool {
	normalized := strings.TrimSpace(strings.ToLower(message))
	confirmations := []string{
		"sim",
		"pode",
		"pode cadastrar",
		"confirmo",
		"confirmar",
		"isso",
		"isso mesmo",
		"correto",
		"esta correto",
		"ta certo",
		"ok",
		"beleza",
	}

	for _, confirmation := range confirmations {
		if normalized == confirmation || strings.Contains(normalized, confirmation) {
			return true
		}
	}
	return false
}

func normalizeText(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	replacements := map[string]string{
		"á": "a", "à": "a", "ã": "a", "â": "a",
		"é": "e", "ê": "e",
		"í": "i",
		"ó": "o", "õ": "o", "ô": "o",
		"ú": "u",
		"ç": "c",
	}
	for from, to := range replacements {
		normalized = strings.ReplaceAll(normalized, from, to)
	}
	return normalized
}

func assistantErrorResponse(err error) (string, string, int) {
	if retryAfter, ok := geminiQuotaInfo(err); ok {
		return quotaReply(retryAfter), "gemini_quota_exceeded", retryAfter
	}

	return "Nao consegui falar com a IA agora. Tente novamente em alguns instantes.", "gemini_unavailable", 0
}

func responseErrorFields(toolResult any) (string, int) {
	result, ok := toolResult.(map[string]any)
	if !ok {
		return "", 0
	}

	errorCode, _ := result["error_code"].(string)
	retryAfter := int(numberToFloat(result["retry_after_seconds"]))
	return errorCode, retryAfter
}

func quotaReply(retryAfter int) string {
	if retryAfter > 0 {
		return "O limite gratuito do Gemini foi atingido agora. Tente novamente em " + strconv.Itoa(retryAfter) + " segundos."
	}
	return "O limite gratuito do Gemini foi atingido agora. Tente novamente em alguns instantes."
}

func localToolReply(toolName string, toolResult any) string {
	result, ok := toolResult.(map[string]any)
	if !ok {
		return "Pronto, consultei os dados para voce."
	}

	if toolName == "create_expense" {
		status, _ := result["status"].(string)
		description, _ := result["description"].(string)
		paymentSource, _ := result["payment_source"].(string)
		date, _ := result["date"].(string)
		amount := numberToFloat(result["amount"])

		if status == "created" {
			return "Cadastrei a despesa " + description + " no valor de R$ " + formatMoney(amount) + ", paga com " + paymentSource + ", na data " + date + "."
		}

		return "Gostaria de cadastrar uma despesa de " + description + ", no valor de R$ " + formatMoney(amount) + ", paga com " + paymentSource + ", na data " + date + ". Posso confirmar?"
	}

	if toolName == "get_monthly_summary" {
		month := numberToFloat(result["month"])
		year := numberToFloat(result["year"])
		totalExpense := numberToFloat(result["total_expense"])
		totalSpentSalary := numberToFloat(result["total_gasto_salario"])
		totalAvailable := numberToFloat(result["total_geral_disponivel"])

		return "Ate agora, em " + strconv.Itoa(int(month)) + "/" + strconv.Itoa(int(year)) +
			", voce gastou R$ " + formatMoney(totalExpense) +
			". Desse total, R$ " + formatMoney(totalSpentSalary) +
			" saiu do salario. Seu saldo geral disponivel esta em R$ " + formatMoney(totalAvailable) + "."
	}

	if toolName == "get_category_expenses" {
		month := numberToFloat(result["month"])
		year := numberToFloat(result["year"])
		totalExpense := numberToFloat(result["total_expense"])
		categoryName, _ := result["category_name"].(string)
		paymentSource, _ := result["payment_source"].(string)

		sourceText := ""
		if strings.TrimSpace(paymentSource) != "" {
			sourceText = " paga com " + paymentSource
		}

		return "Em " + strconv.Itoa(int(month)) + "/" + strconv.Itoa(int(year)) +
			", voce gastou R$ " + formatMoney(totalExpense) +
			" na categoria " + categoryName + sourceText + "."
	}

	if toolName == "list_categories" {
		names, ok := result["names"].([]string)
		if !ok || len(names) == 0 {
			return "Voce ainda nao tem categorias cadastradas."
		}
		return "Voce tem estas categorias cadastradas: " + strings.Join(names, ", ") + "."
	}

	if toolName == "get_all_category_expenses" {
		month := numberToFloat(result["month"])
		year := numberToFloat(result["year"])
		totalExpense := numberToFloat(result["total_expense"])
		items, _ := result["categories"].([]map[string]any)

		if len(items) == 0 {
			return "Nao encontrei gastos por categoria em " + strconv.Itoa(int(month)) + "/" + strconv.Itoa(int(year)) + "."
		}

		parts := make([]string, 0, len(items))
		for _, item := range items {
			categoryName, _ := item["category_name"].(string)
			totalAmount := numberToFloat(item["total_amount"])
			parts = append(parts, categoryName+": R$ "+formatMoney(totalAmount))
		}

		return "Em " + strconv.Itoa(int(month)) + "/" + strconv.Itoa(int(year)) +
			", voce gastou R$ " + formatMoney(totalExpense) +
			" no total. Por categoria: " + strings.Join(parts, "; ") + "."
	}

	return "Pronto, consultei os dados para voce."
}

func numberToFloat(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	default:
		return 0
	}
}

func formatMoney(value float64) string {
	text := strconv.FormatFloat(value, 'f', 2, 64)
	return strings.Replace(text, ".", ",", 1)
}

func systemPrompt() string {
	currentDate := time.Now().Format("2006-01-02")
	return "Voce e o assistente financeiro do App Financeiro. Data atual: " + currentDate + ". " +
		"Responda apenas sobre o aplicativo e sobre os dados financeiros do usuario. " +
		"Para perguntas de como usar o app, explique de forma simples: o usuario pode cadastrar despesas, rendas, categorias, ver resumo mensal, gastos por categoria e graficos. " +
		"Para perguntas financeiras, use as ferramentas disponiveis em vez de inventar numeros. " +
		"Quando o usuario pedir para cadastrar uma despesa com uma categoria, envie o nome exato da categoria em category_name. O backend vai usar a categoria existente ou criar uma nova quando o usuario confirmar. " +
		"Nunca diga que acessou o banco diretamente; quem executa as consultas e o backend Go. " +
		"Para criar, atualizar ou deletar dados, nunca execute sem confirmacao clara do usuario. " +
		"Na ferramenta create_expense, use confirm=false para a primeira interpretacao do pedido. Use confirm=true somente se a mensagem atual confirmar claramente uma proposta anterior. " +
		"Se faltarem mes, ano, valor, descricao ou fonte de pagamento, pergunte antes de chamar ferramenta de cadastro."
}
