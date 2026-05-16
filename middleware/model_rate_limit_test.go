package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMemoryModelRateLimitSuccessLimitIgnoresFailedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalRedisEnabled := common.RedisEnabled
	originalEnabled := setting.ModelRequestRateLimitEnabled
	originalDuration := setting.ModelRequestRateLimitDurationMinutes
	originalTotal := setting.ModelRequestRateLimitCount
	originalSuccess := setting.ModelRequestRateLimitSuccessCount
	originalGroup := setting.ModelRequestRateLimitGroup2JSONString()

	common.RedisEnabled = false
	setting.ModelRequestRateLimitEnabled = true
	setting.ModelRequestRateLimitDurationMinutes = 1
	setting.ModelRequestRateLimitCount = 0
	setting.ModelRequestRateLimitSuccessCount = 1
	require.NoError(t, setting.UpdateModelRequestRateLimitGroupByJSONString(`{}`))
	inMemoryRateLimiter = common.InMemoryRateLimiter{}

	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
		setting.ModelRequestRateLimitEnabled = originalEnabled
		setting.ModelRequestRateLimitDurationMinutes = originalDuration
		setting.ModelRequestRateLimitCount = originalTotal
		setting.ModelRequestRateLimitSuccessCount = originalSuccess
		if err := setting.UpdateModelRequestRateLimitGroupByJSONString(originalGroup); err != nil {
			t.Errorf("restore original rate limit group: %v", err)
		}
		inMemoryRateLimiter = common.InMemoryRateLimiter{}
	})

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", 42)
		c.Next()
	})
	router.Use(ModelRequestRateLimit())
	router.GET("/fail", func(c *gin.Context) {
		c.Status(http.StatusBadRequest)
	})
	router.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	require.Equal(t, http.StatusBadRequest, performModelRateLimitRequest(router, "/fail"))
	require.Equal(t, http.StatusOK, performModelRateLimitRequest(router, "/ok"))
	require.Equal(t, http.StatusTooManyRequests, performModelRateLimitRequest(router, "/ok"))
}

func performModelRateLimitRequest(router http.Handler, target string) int {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Code
}
