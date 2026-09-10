package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
  )

type Studentinfo struct{
	Fullname string `json:"fullname"`
}

func main() {
  router := gin.Default()

  //Get
  router.GET("/student", func(c *gin.Context) {
	studentId := c.Query("studentId")

    c.JSON(200, gin.H{
    "studentId": studentId,
	"fullname": "Jon Doe",
    })
  })

  //Post
  router.POST("/student", func(c *gin.Context) {
	var body Studentinfo

	if err := c.ShouldBindJSON(&body); err != nil{
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      	return
	}
    c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Register successfully %s", body.Fullname)})
  })

  router.Run() // listens on 0.0.0.0:8080 by default

}