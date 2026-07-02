package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestIsOriginAllowed 测试 origin 匹配逻辑（纯函数，无需框架依赖）
func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		allowed []string
		want    bool
	}{
		{"exact match", "https://app.example.com", []string{"https://app.example.com"}, true},
		{"exact mismatch", "https://evil.com", []string{"https://app.example.com"}, false},
		{"wildcard port localhost:3000", "http://localhost:3000", []string{"http://localhost:*"}, true},
		{"wildcard port localhost:8080", "http://localhost:8080", []string{"http://localhost:*"}, true},
		{"wildcard port wrong host", "http://127.0.0.1:3000", []string{"http://localhost:*"}, false},
		{"wildcard port no port in origin", "http://localhost", []string{"http://localhost:*"}, false},
		{"multiple origins first matches", "https://a.com", []string{"https://a.com", "https://b.com"}, true},
		{"multiple origins second matches", "https://b.com", []string{"https://a.com", "https://b.com"}, true},
		{"multiple origins none match", "https://c.com", []string{"https://a.com", "https://b.com"}, false},
		{"empty allowed list", "https://a.com", []string{}, false},
		{"mixed wildcard + exact wildcard hit", "http://localhost:5173", []string{"https://prod.com", "http://localhost:*"}, true},
		{"mixed wildcard + exact exact hit", "https://prod.com", []string{"https://prod.com", "http://localhost:*"}, true},
		{"strict exact trailing slash differs", "https://a.com/", []string{"https://a.com"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOriginAllowed(tt.origin, tt.allowed)
			if got != tt.want {
				t.Errorf("isOriginAllowed(%q, %v) = %v, want %v", tt.origin, tt.allowed, got, tt.want)
			}
		})
	}
}

// TestCORSHandlerIntegration 集成测试：完整 HTTP 请求经过中间件的响应头验证
func TestCORSHandlerIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("no Origin header passes through without CORS headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/ping", nil)

		CORS()(c)

		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("expected no CORS header when Origin is absent")
		}
	})

	t.Run("OPTIONS preflight returns 204 with correct headers", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS())
		r.GET("/api/test", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", w.Code)
		}
		methods := w.Header().Get("Access-Control-Allow-Methods")
		if methods == "" {
			t.Error("expected Access-Control-Allow-Methods header")
		}
	})

	t.Run("allowed Origin gets Allow-Origin on GET request", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")

		r := gin.New()
		r.Use(CORS())
		r.GET("/api/data", func(c *gin.Context) { c.String(200, "ok") })

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		allowed := w.Header().Get("Access-Control-Allow-Origin")
		if allowed != "http://localhost:3000" {
			t.Errorf("expected Allow-Origin=http://localhost:3000, got %q", allowed)
		}
	})
}
