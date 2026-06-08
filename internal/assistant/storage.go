package assistant

import (
	"Sobra_Ai_Back-end/internal/database"
	"encoding/json"
	"strings"
)

func getOrCreateConversation(userID uint, conversationID uint, firstMessage string) (Conversation, error) {
	if conversationID != 0 {
		var conversation Conversation
		err := database.DB.Where("id = ? AND user_id = ?", conversationID, userID).First(&conversation).Error
		return conversation, err
	}

	title := strings.TrimSpace(firstMessage)
	if len(title) > 60 {
		title = title[:60]
	}
	if title == "" {
		title = "Nova conversa"
	}

	conversation := Conversation{
		UserID: userID,
		Title:  title,
	}
	err := database.DB.Create(&conversation).Error
	return conversation, err
}

func saveMessage(userID uint, conversationID uint, role string, content string, toolCall string, toolResult any) error {
	var toolResultJSON string
	if toolResult != nil {
		resultBytes, err := json.Marshal(toolResult)
		if err != nil {
			return err
		}
		toolResultJSON = string(resultBytes)
	}

	message := Message{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           role,
		Content:        strings.TrimSpace(content),
		ToolCall:       toolCall,
		ToolResult:     toolResultJSON,
	}

	if err := database.DB.Create(&message).Error; err != nil {
		return err
	}

	return database.DB.Model(&Conversation{}).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Update("updated_at", message.CreatedAt).Error
}

func loadRecentMessages(userID uint, conversationID uint, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 20
	}

	var newest []Message
	err := database.DB.
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Order("created_at desc").
		Limit(limit).
		Find(&newest).Error
	if err != nil {
		return nil, err
	}

	messages := make([]Message, len(newest))
	for i := range newest {
		messages[len(newest)-1-i] = newest[i]
	}
	return messages, nil
}

func buildConversationFromMessages(messages []Message, currentMessage string) []GeminiContent {
	contents := make([]GeminiContent, 0, len(messages)+1)
	for _, message := range messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		role := "user"
		if message.Role == "assistant" || message.Role == "model" {
			role = "model"
		}

		contents = append(contents, GeminiContent{
			Role:  role,
			Parts: []GeminiPart{{Text: content}},
		})
	}

	contents = append(contents, GeminiContent{
		Role:  "user",
		Parts: []GeminiPart{{Text: strings.TrimSpace(currentMessage)}},
	})

	return contents
}

func lastPendingCreateExpense(userID uint, conversationID uint) (map[string]any, bool) {
	var messages []Message
	err := database.DB.
		Where("conversation_id = ? AND user_id = ? AND role = ? AND tool_call = ?", conversationID, userID, "assistant", "create_expense").
		Order("created_at desc").
		Limit(1).
		Find(&messages).Error
	if err != nil || len(messages) == 0 || messages[0].ToolResult == "" {
		return nil, false
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(messages[0].ToolResult), &result); err != nil {
		return nil, false
	}

	status, _ := result["status"].(string)
	if status != "needs_confirmation" {
		return nil, false
	}

	return result, true
}
