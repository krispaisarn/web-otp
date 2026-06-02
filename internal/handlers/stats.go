package handlers

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/krispaisarn/web-otp/internal/db"
	"github.com/krispaisarn/web-otp/internal/models"
)

type summaryStats struct {
	TotalIssued   int64 `json:"total_issued"`
	TotalVerified int64 `json:"total_verified"`
	TotalExpired  int64 `json:"total_expired"`
	TotalPending  int64 `json:"total_pending"`
}

func Stats(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := c.QueryInt("offset", 0)
	if offset < 0 {
		offset = 0
	}

	database, err := db.Get()
	if err != nil {
		log.Printf("ERROR Stats db: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "database unavailable")
	}

	base := database.WithContext(c.Context()).Model(&models.OTP{})
	if from := c.Query("from"); from != "" {
		base = base.Where("created_at >= ?", from)
	}
	if to := c.Query("to"); to != "" {
		base = base.Where("created_at <= ?", to)
	}
	if em := c.Query("email"); em != "" {
		base = base.Where("email LIKE ?", "%"+em+"%")
	}

	var summary summaryStats
	now := time.Now()

	base.Count(&summary.TotalIssued)
	base.Where("used = ?", true).Count(&summary.TotalVerified)
	base.Where("used = ? AND expires_at < ?", false, now).Count(&summary.TotalExpired)
	base.Where("used = ? AND expires_at >= ?", false, now).Count(&summary.TotalPending)

	var records []models.OTP
	if err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		log.Printf("ERROR Stats query: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "records query failed")
	}

	return c.JSON(fiber.Map{
		"summary": summary,
		"records": records,
		"total":   summary.TotalIssued,
	})
}
