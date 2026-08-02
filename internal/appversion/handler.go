package appversion

import (
	"Sobra_Ai_Back-end/internal/auth"
	"Sobra_Ai_Back-end/internal/database"
	"Sobra_Ai_Back-end/internal/pagination"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/app-version", getAppVersion)

	adminGroup := rg.Group("/admin")
	adminGroup.Use(auth.AuthMiddleware(), auth.AdminMiddleware())
	{
		adminGroup.POST("/app-version/:platform", createAppVersion)
		adminGroup.PATCH("/app-version/:platform/:version_code", updateAppVersion)
		adminGroup.GET("/app-version/:platform/history", listAppVersions)
	}
}

func getAppVersion(c *gin.Context) {
	platform := normalizePlatform(c.DefaultQuery("platform", "android"))
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Plataforma invalida"})
		return
	}

	var version AppVersion
	if err := database.DB.
		Where("platform = ?", platform).
		Order("latest_version_code desc, created_at desc").
		First(&version).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Versao do app nao cadastrada"})
		return
	}

	c.JSON(http.StatusOK, appVersionResponse(version))
}

func createAppVersion(c *gin.Context) {
	platform := normalizePlatform(c.Param("platform"))
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Plataforma invalida"})
		return
	}

	var req SaveAppVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados invalidos: " + err.Error()})
		return
	}

	version, ok := buildAppVersion(c, platform, req)
	if !ok {
		return
	}

	var existing AppVersion
	err := database.DB.
		Where("platform = ? AND latest_version_code = ?", platform, version.LatestVersionCode).
		First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Versao do app ja cadastrada para esta plataforma"})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar versao do app"})
		return
	}

	if err := database.DB.Create(&version).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao cadastrar versao do app"})
		return
	}

	c.JSON(http.StatusCreated, appVersionResponse(version))
}

func updateAppVersion(c *gin.Context) {
	platform := normalizePlatform(c.Param("platform"))
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Plataforma invalida"})
		return
	}

	versionCode, err := strconv.Atoi(c.Param("version_code"))
	if err != nil || versionCode <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Codigo da versao invalido"})
		return
	}

	var req SaveAppVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados invalidos: " + err.Error()})
		return
	}

	version, ok := buildAppVersion(c, platform, req)
	if !ok {
		return
	}

	if version.LatestVersionCode != versionCode {
		c.JSON(http.StatusBadRequest, gin.H{"error": "latest_version_code deve ser igual ao codigo informado na URL"})
		return
	}

	var existing AppVersion
	if err := database.DB.
		Where("platform = ? AND latest_version_code = ?", platform, versionCode).
		First(&existing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Versao do app nao cadastrada"})
		return
	}

	if err := database.DB.Model(&existing).Updates(map[string]interface{}{
		"latest_version_name":       version.LatestVersionName,
		"min_required_version_code": version.MinRequiredVersionCode,
		"force_update":              version.ForceUpdate,
		"play_store_url":            version.PlayStoreURL,
		"message":                   version.Message,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar versao do app"})
		return
	}

	if err := database.DB.
		Where("platform = ? AND latest_version_code = ?", platform, versionCode).
		First(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar versao atualizada"})
		return
	}

	c.JSON(http.StatusOK, appVersionResponse(existing))
}

func listAppVersions(c *gin.Context) {
	platform := normalizePlatform(c.Param("platform"))
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Plataforma invalida"})
		return
	}
	params, err := pagination.Parse(c.Request.URL.Query())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var versions []AppVersion
	var total int64
	query := database.DB.Where("platform = ?", platform)
	if params.Enabled {
		if err := database.DB.Model(&AppVersion{}).Where("platform = ?", platform).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao contar versoes"})
			return
		}
		query = query.Limit(params.Limit).Offset(params.Offset)
	}
	if err := query.Order("latest_version_code desc, created_at desc").Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar historico de versoes"})
		return
	}

	response := make([]AppVersionResponse, 0, len(versions))
	for _, version := range versions {
		response = append(response, appVersionResponse(version))
	}

	if !params.Enabled {
		c.JSON(http.StatusOK, gin.H{"total": len(response), "versions": response})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"total":      total,
		"versions":   response,
		"pagination": pagination.NewMetadata(params, total),
	})
}

func buildAppVersion(c *gin.Context, platform string, req SaveAppVersionRequest) (AppVersion, bool) {
	version := AppVersion{
		Platform:               platform,
		LatestVersionName:      strings.TrimSpace(req.LatestVersionName),
		LatestVersionCode:      req.LatestVersionCode,
		MinRequiredVersionCode: req.MinRequiredVersionCode,
		ForceUpdate:            req.ForceUpdate,
		PlayStoreURL:           strings.TrimSpace(req.PlayStoreURL),
		Message:                strings.TrimSpace(req.Message),
	}

	if version.LatestVersionName == "" || version.PlayStoreURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "latest_version_name e play_store_url sao obrigatorios"})
		return version, false
	}

	if version.MinRequiredVersionCode > version.LatestVersionCode {
		c.JSON(http.StatusBadRequest, gin.H{"error": "min_required_version_code nao pode ser maior que latest_version_code"})
		return version, false
	}

	return version, true
}

func normalizePlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "android":
		return "android"
	default:
		return ""
	}
}

func appVersionResponse(version AppVersion) AppVersionResponse {
	return AppVersionResponse{
		ID:                     version.ID,
		Platform:               version.Platform,
		LatestVersionName:      version.LatestVersionName,
		LatestVersionCode:      version.LatestVersionCode,
		MinRequiredVersionCode: version.MinRequiredVersionCode,
		ForceUpdate:            version.ForceUpdate,
		PlayStoreURL:           version.PlayStoreURL,
		Message:                version.Message,
		CreatedAt:              version.CreatedAt,
		UpdatedAt:              version.UpdatedAt,
	}
}
