package auth

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUserMetricsResponseJSONContract(t *testing.T) {
	response := UserMetricsResponse{Total: 16, Active: 15, Blocked: 1, Admins: 1}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal metrics: %v", err)
	}

	const expected = `{"total":16,"active":15,"blocked":1,"admins":1}`
	if string(data) != expected {
		t.Fatalf("expected %s, got %s", expected, data)
	}
}

func TestRegisterRoutesIncludesAdminUserMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api"))

	for _, route := range router.Routes() {
		if route.Method == http.MethodGet && route.Path == "/api/admin/users/metrics" {
			return
		}
	}

	t.Fatal("GET /api/admin/users/metrics was not registered")
}

func TestRegisterRoutesIncludesPremiumPromotion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api"))

	for _, route := range router.Routes() {
		if route.Method == http.MethodPatch && route.Path == "/api/admin/users/:id/promote-premium" {
			return
		}
	}

	t.Fatal("PATCH /api/admin/users/:id/promote-premium was not registered")
}

func TestRegisterRoutesIncludesPremiumRevocation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api"))

	for _, route := range router.Routes() {
		if route.Method == http.MethodPatch && route.Path == "/api/admin/users/:id/revoke-premium" {
			return
		}
	}

	t.Fatal("PATCH /api/admin/users/:id/revoke-premium was not registered")
}
