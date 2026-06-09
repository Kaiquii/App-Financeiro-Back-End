package users

import (
	"Sobra_Ai_Back-end/internal/auth"
	"Sobra_Ai_Back-end/internal/database"
	"Sobra_Ai_Back-end/internal/uploads"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	avatarSize        = 512
	maxAvatarBytes    = 5 * 1024 * 1024
	avatarJPEGQuality = 85
)

func RegisterRoutes(rg *gin.RouterGroup) {
	usersGroup := rg.Group("/users")
	{
		usersGroup.GET("/profile", getProfile)
		usersGroup.PATCH("/profile", updateProfile)
		usersGroup.PATCH("/profile/photo", updateProfilePhoto)
		usersGroup.DELETE("/profile/photo", deleteProfilePhoto)
	}
}

func getProfile(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario nao identificado"})
		return
	}

	var user auth.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario nao encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": profileResponse(user)})
}

func updateProfile(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario nao identificado"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados invalidos"})
		return
	}

	updateData := make(map[string]interface{})
	if req.Name != "" {
		updateData["name"] = req.Name
	}
	if req.Email != "" {
		updateData["email"] = req.Email
	}

	if err := database.DB.Model(&auth.User{}).Where("id = ?", userID).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar perfil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Perfil atualizado com sucesso!"})
}

func updateProfilePhoto(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario nao identificado"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAvatarBytes+1024*1024)

	fileHeader, err := c.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Envie a foto no campo photo"})
		return
	}

	if fileHeader.Size > maxAvatarBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A foto deve ter no maximo 5 MB"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao abrir foto enviada"})
		return
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil || !isAllowedAvatarFormat(format) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Envie uma foto JPG, JPEG, PNG ou GIF valida"})
		return
	}

	avatar := resizeCenterCrop(img, avatarSize)
	userIDText := strconv.FormatUint(uint64(userID), 10)
	userDir := filepath.Join(uploads.Dir(), "users", userIDText)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao preparar pasta da foto"})
		return
	}

	avatarPath := filepath.Join(userDir, "avatar.jpg")
	output, err := os.Create(avatarPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar foto"})
		return
	}
	defer output.Close()

	if err := jpeg.Encode(output, avatar, &jpeg.Options{Quality: avatarJPEGQuality}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao processar foto"})
		return
	}

	avatarURL := uploads.PublicURL("users", userIDText, "avatar.jpg")
	if err := database.DB.Model(&auth.User{}).Where("id = ?", userID).Update("avatar_url", avatarURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar foto do perfil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Foto de perfil atualizada com sucesso!",
		"avatar_url": avatarURL,
	})
}

func deleteProfilePhoto(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario nao identificado"})
		return
	}

	userIDText := strconv.FormatUint(uint64(userID), 10)
	avatarPath := filepath.Join(uploads.Dir(), "users", userIDText, "avatar.jpg")
	if err := os.Remove(avatarPath); err != nil && !os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao remover foto"})
		return
	}

	if err := database.DB.Model(&auth.User{}).Where("id = ?", userID).Update("avatar_url", "").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar perfil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Foto de perfil removida com sucesso!"})
}

func currentUserID(c *gin.Context) (uint, bool) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	userID, ok := userIDObj.(uint)
	return userID, ok
}

func profileResponse(user auth.User) gin.H {
	return gin.H{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"role":       user.Role,
		"avatar_url": user.AvatarURL,
	}
}

func isAllowedAvatarFormat(format string) bool {
	switch format {
	case "jpeg", "png", "gif":
		return true
	default:
		return false
	}
}

func resizeCenterCrop(src image.Image, size int) *image.RGBA {
	bounds := src.Bounds()
	side := bounds.Dx()
	if bounds.Dy() < side {
		side = bounds.Dy()
	}

	startX := bounds.Min.X + (bounds.Dx()-side)/2
	startY := bounds.Min.Y + (bounds.Dy()-side)/2
	dst := image.NewRGBA(image.Rect(0, 0, size, size))

	for y := 0; y < size; y++ {
		sourceY := startY + y*side/size
		for x := 0; x < size; x++ {
			sourceX := startX + x*side/size
			dst.Set(x, y, flattenOnWhite(src.At(sourceX, sourceY)))
		}
	}

	return dst
}

func flattenOnWhite(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	if a == 0xffff {
		return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}
	}

	alpha := float64(a) / 65535
	red := uint8((float64(r>>8) * alpha) + (255 * (1 - alpha)))
	green := uint8((float64(g>>8) * alpha) + (255 * (1 - alpha)))
	blue := uint8((float64(b>>8) * alpha) + (255 * (1 - alpha)))

	return color.RGBA{R: red, G: green, B: blue, A: 255}
}
