package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Studentinfo struct {
	Fullname string `json:"fullname"`
}

func main() {
	router := gin.Default()

	// Serve the page from Go, so the browser sees the page and the API as one origin.
	// Path is relative to where you run the program: cd server && go run .
	router.StaticFile("/", "../web-app/index.html")

	//Get
	router.GET("/student", func(c *gin.Context) {
		studentId := c.Query("studentId")

		c.JSON(200, gin.H{
			"studentId": studentId,
			"name":      "Jon Doe",
			"email":     "sample@mail.com",
		})
	})

	//Post
	router.POST("/student", func(c *gin.Context) {
		var body Studentinfo

		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"studentId": "1234",
			"name":      "Jon Doe",
			"email":     "sample@mail.com",
		})
	})

	router.Run() // listens on 0.0.0.0:8080 by default

}
