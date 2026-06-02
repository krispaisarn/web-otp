package otp

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/krispaisarn/web-otp/internal/db"
	"github.com/krispaisarn/web-otp/internal/models"
	"gorm.io/gorm"
)

// Issue generates an OTP for email, binding it to sessionToken.
// The same sessionToken must be presented on Verify.
func Issue(ctx context.Context, email, sessionToken string) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("generating OTP: %w", err)
	}
	code := fmt.Sprintf("%06d", n.Int64())

	database, err := db.Get()
	if err != nil {
		return "", err
	}

	record := &models.OTP{
		Email:        email,
		Code:         code,
		SessionToken: sessionToken,
		ExpiresAt:    time.Now().UTC().Add(time.Hour),
	}
	if err := database.WithContext(ctx).Create(record).Error; err != nil {
		return "", fmt.Errorf("inserting OTP: %w", err)
	}

	return code, nil
}

// Verify checks the OTP for email, requiring the same sessionToken used on Issue.
// Returns true and marks the OTP used on success.
func Verify(ctx context.Context, email, code, sessionToken string) (bool, error) {
	database, err := db.Get()
	if err != nil {
		return false, err
	}

	var record models.OTP
	err = database.WithContext(ctx).
		Where("email = ? AND otp = ? AND session_token = ? AND used = ? AND expires_at > ?",
			email, code, sessionToken, false, time.Now()).
		First(&record).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("querying OTP: %w", err)
	}

	now := time.Now()
	if err := database.WithContext(ctx).Model(&record).
		Updates(map[string]any{"used": true, "used_at": &now}).Error; err != nil {
		return false, fmt.Errorf("marking OTP used: %w", err)
	}

	return true, nil
}
