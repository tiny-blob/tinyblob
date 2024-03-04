package routes

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/tiny-blob/tinyblob/config"
	"github.com/tiny-blob/tinyblob/pkg/controller"
	mw "github.com/tiny-blob/tinyblob/pkg/middleware"
	"github.com/tiny-blob/tinyblob/pkg/services"
)

// BuildRouter builds the router.
func BuildRouter(c *services.Container) {
	// Enable cache control for static files.
	// NOTE: We need to use funcmap.File() to append a cache key to the URL in
	// order to break the cache after each server restart.
	c.Web.Group("", mw.CacheControl(c.Config.Cache.Expiration.StaticFile)).
		Static(config.StaticPrefix, config.StaticDir)

	// Non-static routes
	g := c.Web.Group("")

	// Force HTTPS if enabled
	if c.Config.HTTP.TLS.Enabled {
		g.Use(echomw.HTTPSRedirect())
	}

	g.Use(
		echomw.RemoveTrailingSlashWithConfig(echomw.TrailingSlashConfig{
			RedirectCode: http.StatusMovedPermanently,
		}),
		echomw.RecoverWithConfig(echomw.RecoverConfig{
			StackSize: 1 << 10, // 1 KB
			LogLevel:  log.ERROR,
		}),
		echomw.Secure(),
		echomw.RequestID(),
		mw.LogRequestID(),
		// TODO: See https://github.com/labstack/echo/issues/1223
		mw.Brotli(),
		// TODO: https://betterstack.com/community/guides/logging/go/zap/
		echomw.Logger(),
		echomw.TimeoutWithConfig(echomw.TimeoutConfig{
			Timeout: c.Config.App.Timeout,
		}),
		session.Middleware(sessions.NewCookieStore([]byte(c.Config.App.EncryptionKey))),
		// mw.ServeCachedPage(c.Cache),
		echomw.CSRFWithConfig(echomw.CSRFConfig{
			TokenLookup:    "form:csrf",
			CookieSameSite: http.SameSiteStrictMode,
		}),
	)

	// Base controller
	ctr := controller.NewController(c)

	// Global error handler
	err := errorHandler{Controller: ctr}
	c.Web.HTTPErrorHandler = err.Get

	// Routes
	publicRoutes(c, g, ctr)
}

func publicRoutes(c *services.Container, g *echo.Group, ctr controller.Controller) {
	home := home{Controller: ctr}
	g.GET("/", home.Get).Name = "home"
}
