package main

import (
	"github.com/gin-gonic/gin"
)

func registerRoutes(router *gin.Engine, app *App) {
	// Handlers for shared content.
	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/img/favicon.ico")

	// Dashboard routes.
	router.GET("/", homeHandler)

	// Healthcheck route.
	router.GET("/health", func(c *gin.Context) {
		healthcheckHandler(app, c)
	})

	// Projects routes.
	router.GET("/projects", func(c *gin.Context) {
		projectsHandler(app, c)
	})
	router.POST("/add-project", func(c *gin.Context) {
		addProjectHandler(app, c)
	})
	router.DELETE("/delete-project", func(c *gin.Context) {
		deleteProjectHandler(app, c)
	})
	router.POST("/update-project", func(c *gin.Context) {
		editProjectHandler(app, c)
	})
}
