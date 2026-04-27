package server

import (
	"log"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func StartServer(mode string) {
	var router *echo.Echo
	switch mode {
	case "WEB_SERVER":
		router = newRouter()
		log.Println("Router Initialized")
	case "ARGO_PLUGIN":
		return
	default:
		log.Println("UNKNOWN MODE! ABORT")
	}

	if router == nil {
		log.Println("Router Not Initialized")
		return
	}
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.RequestLogger())
	e.Any("/*", func(c *echo.Context) (err error) {
		req := c.Request()
		resp := c.Response()
		router.ServeHTTP(resp, req)
		return
	})
	e.Start(":8080")
}
