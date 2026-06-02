package app

import (
	"errors"
	"log"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/krispaisarn/web-otp/internal/handlers"
)

var (
	instance *fiber.App
	once     sync.Once
)

func Get() *fiber.App {
	once.Do(func() {
		instance = fiber.New(fiber.Config{
			ErrorHandler: errorHandler,
		})

		instance.Use(cors.New())

		api := instance.Group("/api")
		api.Post("/otp", handlers.IssueOTP)
		api.Post("/otp/verify", handlers.VerifyOTP)
		api.Get("/stats", handlers.Stats)
	})
	return instance
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "internal server error"
	var fe *fiber.Error
	if errors.As(err, &fe) {
		code = fe.Code
		msg = fe.Message
	} else {
		log.Printf("ERROR unhandled: %v", err)
	}
	return c.Status(code).JSON(fiber.Map{"error": msg})
}
