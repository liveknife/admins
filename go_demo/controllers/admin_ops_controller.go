package controllers

import (
	"net/http"
	"strings"

	"go-demo/services"

	"github.com/gin-gonic/gin"
)

type SystemSettingRequest struct {
	SettingKey   string `json:"setting_key" form:"setting_key"`
	SettingValue string `json:"setting_value" form:"setting_value"`
	GroupName    string `json:"group_name" form:"group_name"`
	ValueType    string `json:"value_type" form:"value_type"`
	Description  string `json:"description" form:"description"`
	IsSecret     bool   `json:"is_secret" form:"is_secret"`
}

type SystemSettingsRequest struct {
	Settings []SystemSettingRequest `json:"settings" form:"settings"`
}

func (c *AdminController) ListSystemSettings(g *gin.Context) {
	items, err := c.adminData.ListSystemSettings(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"settings": items})
}

func (c *AdminController) SaveSystemSettings(g *gin.Context) {
	var req SystemSettingsRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	inputs := make([]services.SystemSettingInput, 0, len(req.Settings))
	for _, item := range req.Settings {
		inputs = append(inputs, services.SystemSettingInput{
			SettingKey:   strings.TrimSpace(item.SettingKey),
			SettingValue: strings.TrimSpace(item.SettingValue),
			GroupName:    strings.TrimSpace(item.GroupName),
			ValueType:    strings.TrimSpace(item.ValueType),
			Description:  strings.TrimSpace(item.Description),
			IsSecret:     item.IsSecret,
		})
	}
	items, err := c.adminData.SaveSystemSettings(g.Request.Context(), inputs)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save system settings"})
		return
	}
	c.logAction(g, "保存系统配置", "系统配置中心", "settings")
	g.JSON(http.StatusOK, gin.H{"settings": items})
}

func (c *AdminController) ListAIModelCallLogs(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	items, total, err := c.adminData.ListAIModelCallLogs(g.Request.Context(), page, pageSize, g.Query("provider"), g.Query("operation"), g.Query("status"))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list ai model call logs"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"logs": items, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) AIModelCallStats(g *gin.Context) {
	stats, err := c.adminData.AIModelCallStats(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ai model call stats"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (c *AdminController) SiteOperationsDashboard(g *gin.Context) {
	dashboard, err := c.adminData.SiteOperationsDashboard(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load site operations dashboard"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"dashboard": dashboard})
}
