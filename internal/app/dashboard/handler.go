package dashboard

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var loc = time.FixedZone("America/Sao_Paulo", -3*60*60)

type DashboardHandler struct {
	Service DashboardServiceI
}

func NewDashboardHandler(service DashboardServiceI) *DashboardHandler {
	return &DashboardHandler{Service: service}
}

func parseStartDate(val string, defaultDaysAgo int) time.Time {
	if val != "" {
		t, err := time.ParseInLocation("2006-01-02", val, loc)
		if err == nil {
			return t.UTC()
		}
	}
	nowLocal := time.Now().In(loc)
	startLocal := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day()-defaultDaysAgo, 0, 0, 0, 0, loc)
	return startLocal.UTC()
}

func parseEndDate(val string) time.Time {
	if val != "" {
		t, err := time.ParseInLocation("2006-01-02", val, loc)
		if err == nil {
			endLocal := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, loc)
			return endLocal.UTC()
		}
	}
	return time.Now().UTC()
}

func (h *DashboardHandler) GetStats(c *gin.Context) {
	startStr := c.Query("createdAt_start")
	endStr := c.Query("createdAt_end")

	start := parseStartDate(startStr, 30)
	end := parseEndDate(endStr)

	if start.After(end) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "A data de início deve ser anterior ou igual à data de fim",
		})
		return
	}

	stats, err := h.Service.GetStats(c.Request.Context(), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
