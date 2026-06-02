package models

import "time"

type OTP struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Email        string     `gorm:"size:255;not null;index:idx_email" json:"email"`
	Code         string     `gorm:"column:otp;size:6;not null" json:"-"`
	SessionToken string     `gorm:"size:128;not null;index" json:"-"`
	ExpiresAt    time.Time  `gorm:"not null;index" json:"expires_at"`
	Used         bool       `gorm:"not null;default:false" json:"used"`
	UsedAt       *time.Time `gorm:"index" json:"used_at"`
	CreatedAt    time.Time  `gorm:"index" json:"created_at"`
}

func (OTP) TableName() string { return "otps" }
