package models

import "time"

type OTP struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement"`
	Email        string     `gorm:"size:255;not null;index:idx_email"`
	Code         string     `gorm:"column:otp;size:6;not null"`
	SessionToken string     `gorm:"size:128;not null;index"`
	ExpiresAt    time.Time  `gorm:"not null;index"`
	Used         bool       `gorm:"not null;default:false"`
	UsedAt       *time.Time `gorm:"index"`
	CreatedAt    time.Time  `gorm:"index"`
}

func (OTP) TableName() string { return "otps" }
