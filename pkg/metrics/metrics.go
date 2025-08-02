package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	invalidTokenRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "invalid_token_requests_total",
			Help: "无效token请求总次数",
		},
		[]string{"token"},
	)

	userSubscribeRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_subscribe_requests_total",
			Help: "用户订阅获取总次数",
		},
		[]string{"username", "subscribe_group", "subscribe_name"},
	)

	cacheUpdateRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_update_requests_total",
			Help: "缓存更新总次数",
		},
		[]string{"subscribe_group", "subscribe_name"},
	)

	cacheUpdateSuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_update_success_total",
			Help: "缓存更新成功总次数",
		},
		[]string{"subscribe_group", "subscribe_name"},
	)
)

func GetMetrics() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}

func RecordInvalidToken(token string) {
	invalidTokenRequests.WithLabelValues(token).Inc()
}

func RecordUserSubscribe(username, subscribeGroup, subscribeName string) {
	userSubscribeRequests.WithLabelValues(username, subscribeGroup, subscribeName).Inc()
}

func RecordCacheUpdate(subscribeGroup, subscribeName string) {
	cacheUpdateRequests.WithLabelValues(subscribeGroup, subscribeName).Inc()
}

func RecordCacheUpdateSuccess(subscribeGroup, subscribeName string) {
	cacheUpdateSuccess.WithLabelValues(subscribeGroup, subscribeName).Inc()
}
