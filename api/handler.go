package handler

import (
	"net/http"

	"github.com/gofiber/adaptor/v2"
	"github.com/krispaisarn/web-otp/internal/app"
)

// Handler is the Vercel serverless entry point.
// adaptor.FiberApp bridges the Fiber app to the net/http interface Vercel expects.
var fiberHandler = adaptor.FiberApp(app.Get())

func Handler(w http.ResponseWriter, r *http.Request) {
	fiberHandler(w, r)
}
