package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/krispaisarn/web-otp/internal/otp"
)

type verifyRequest struct {
	Email        string `json:"email"`
	OTP          string `json:"otp"`
	SessionToken string `json:"session_token"`
}

func VerifyOTP(c *fiber.Ctx) error {
	var req verifyRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" || req.OTP == "" || req.SessionToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email, otp, and session_token are required")
	}

	valid, err := otp.Verify(c.Context(), req.Email, req.OTP, req.SessionToken)
	if err != nil {
		log.Printf("ERROR VerifyOTP: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to verify OTP")
	}

	return c.JSON(fiber.Map{"valid": valid})
}
