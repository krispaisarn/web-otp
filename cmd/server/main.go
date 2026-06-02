// server runs the OTP API locally, loading environment variables from .env.
//
// Usage:
//
//	go run ./cmd/server
//	PORT=9000 go run ./cmd/server
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/krispaisarn/web-otp/internal/app"
)

func main() {
	loadDotEnv(".env")

	fiberApp := app.Get()

	fiberApp.Get("/view", func(c *fiber.Ctx) error {
		return c.SendFile("public/view.html")
	})
	fiberApp.Get("/docs", func(c *fiber.Ctx) error {
		return c.SendFile("public/docs.html")
	})
	fiberApp.Static("/", "public")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("Listening on http://localhost:" + port)
	fmt.Println("  POST http://localhost:" + port + "/api/otp")
	fmt.Println("  POST http://localhost:" + port + "/api/otp/verify")
	fmt.Println("  GET  http://localhost:" + port + "/api/stats")
	fmt.Println("  GET  http://localhost:" + port + "/view")
	fmt.Println("  GET  http://localhost:" + port + "/docs")

	log.Fatal(fiberApp.Listen(":" + port))
}

func loadDotEnv(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') ||
			(val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
