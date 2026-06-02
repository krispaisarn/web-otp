package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/krispaisarn/web-otp/internal/email"
	"github.com/krispaisarn/web-otp/internal/otp"
)

type issueRequest struct {
	Email        string `json:"email"`
	SessionToken string `json:"session_token"`
}

func IssueOTP(c *fiber.Ctx) error {
	var req issueRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if !emailRE.MatchString(req.Email) {
		return fiber.NewError(fiber.StatusBadRequest, "valid email is required")
	}
	if len(req.SessionToken) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "session_token is required (min 8 chars)")
	}

	code, err := otp.Issue(c.Context(), req.Email, req.SessionToken)
	if err != nil {
		log.Printf("ERROR IssueOTP: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to issue OTP")
	}

	if err := email.SendOTP(c.Context(), req.Email, code); err != nil {
		log.Printf("ERROR SendOTP: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to send OTP email")
	}

	return c.JSON(fiber.Map{"message": "OTP sent to your email"})
}
