package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func homeHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index", gin.H{
		"Title":       "Dashboard",
		"CurrentPage": "home",
	})
}
