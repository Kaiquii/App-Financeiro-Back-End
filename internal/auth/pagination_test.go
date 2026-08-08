package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetUsersRejectsInvalidPaginationBeforeQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/auth/users?limit=101", nil)

	getUsers(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestParseUserFilters(t *testing.T) {
	filters, err := parseUserFilters(url.Values{
		"search": {" Maria "},
		"role":   {" ADMIN "},
		"status": {" active "},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters.Search != "Maria" || filters.Role != "admin" {
		t.Fatalf("unexpected filters: %+v", filters)
	}
	if !filters.StatusEnabled || filters.Blocked {
		t.Fatalf("expected active status filter, got %+v", filters)
	}
}

func TestParseUserFiltersAcceptsPremiumRole(t *testing.T) {
	filters, err := parseUserFilters(url.Values{"role": {"premium"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters.Role != "premium" {
		t.Fatalf("expected premium role filter, got %q", filters.Role)
	}
}

func TestGetUsersRejectsInvalidFiltersBeforeQuery(t *testing.T) {
	tests := []string{
		"/api/auth/users?role=manager",
		"/api/auth/users?status=pending",
	}

	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodGet, target, nil)

			getUsers(context)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", recorder.Code)
			}
		})
	}
}
