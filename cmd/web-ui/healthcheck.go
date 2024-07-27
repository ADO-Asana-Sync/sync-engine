package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func healthcheckHandler(app *App, c *gin.Context) {
	_, span := app.Tracer.Start(c.Request.Context(), "healthcheck")
	defer span.End()
	c.JSON(200, gin.H{
		"status":  "ok",
		"version": fmt.Sprintf("version: %v, commit: %v, date: %v", Version, Commit, Date),
	})
}
