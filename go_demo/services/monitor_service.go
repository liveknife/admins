package services

import (
	"database/sql"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	"go-demo/models"
)

type RuntimeStatsProvider interface {
	WebSocketStats() (int, int)
}

type MonitorService struct {
	db      *sql.DB
	stats   RuntimeStatsProvider
	mu      sync.RWMutex
	metrics map[string]*apiMetric
	total   int64
}

type apiMetric struct {
	Method      string
	Path        string
	Count       int64
	TotalMS     int64
	SlowCount   int64
	StatusCodes map[int]int64
}

func NewMonitorService(db *sql.DB) *MonitorService {
	return &MonitorService{db: db, metrics: make(map[string]*apiMetric)}
}

func (s *MonitorService) SetRuntimeStatsProvider(stats RuntimeStatsProvider) {
	s.stats = stats
}

func (s *MonitorService) RecordRequest(method, path string, status int, duration time.Duration) {
	if s == nil {
		return
	}
	ms := duration.Milliseconds()
	key := method + " " + path
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.metrics[key]
	if item == nil {
		item = &apiMetric{Method: method, Path: path, StatusCodes: make(map[int]int64)}
		s.metrics[key] = item
	}
	item.Count++
	item.TotalMS += ms
	item.StatusCodes[status]++
	if ms >= 800 {
		item.SlowCount++
	}
	s.total++
}

func (s *MonitorService) Health() models.SystemHealth {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	dbHealth := s.databaseHealth()
	api := s.apiHealth()
	wsUsers, wsConnections := 0, 0
	if s.stats != nil {
		wsUsers, wsConnections = s.stats.WebSocketStats()
	}

	health := models.SystemHealth{
		Status: "healthy",
		CPU: models.HealthMetric{
			Value: float64(runtime.NumGoroutine()),
			Unit:  "goroutine",
			Label: "运行协程",
		},
		Memory: models.HealthMetric{
			Value: float64(mem.Alloc) / 1024 / 1024,
			Unit:  "MB",
			Label: "当前内存",
		},
		Database:  dbHealth,
		WebSocket: models.WebSocketHealth{OnlineUsers: wsUsers, Connections: wsConnections},
		API:       api,
		CheckedAt: time.Now(),
	}

	if dbHealth.Status != "ok" {
		health.Alerts = append(health.Alerts, "数据库连接异常")
	}
	if health.Memory.Value > 512 {
		health.Alerts = append(health.Alerts, "内存占用偏高")
	}
	if api.SlowRequests > 0 {
		health.Alerts = append(health.Alerts, "存在慢接口请求")
	}
	if len(health.Alerts) > 0 {
		health.Status = "warning"
	}
	return health
}

func (s *MonitorService) databaseHealth() models.DatabaseHealth {
	start := time.Now()
	status := "ok"
	if err := s.db.Ping(); err != nil {
		status = "error"
	}
	stat := s.db.Stats()
	return models.DatabaseHealth{
		Status:         status,
		OpenConnection: stat.OpenConnections,
		InUse:          stat.InUse,
		Idle:           stat.Idle,
		WaitCount:      stat.WaitCount,
		PingMS:         time.Since(start).Milliseconds(),
	}
}

func (s *MonitorService) apiHealth() models.APIHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := models.APIHealth{
		TotalRequests: s.total,
		StatusCodes:   make(map[string]int64),
	}
	var totalMS int64
	items := make([]models.APIMetricSummary, 0, len(s.metrics))
	for _, item := range s.metrics {
		totalMS += item.TotalMS
		out.SlowRequests += item.SlowCount
		for code, count := range item.StatusCodes {
			out.StatusCodes[strconv.Itoa(code)] += count
		}
		avg := 0.0
		if item.Count > 0 {
			avg = float64(item.TotalMS) / float64(item.Count)
		}
		items = append(items, models.APIMetricSummary{
			Path:      item.Path,
			Method:    item.Method,
			Count:     item.Count,
			AverageMS: avg,
		})
	}
	if s.total > 0 {
		out.AverageMS = float64(totalMS) / float64(s.total)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].AverageMS == items[j].AverageMS {
			return items[i].Count > items[j].Count
		}
		return items[i].AverageMS > items[j].AverageMS
	})
	if len(items) > 8 {
		items = items[:8]
	}
	out.TopPaths = items
	return out
}
