package main

import (
  
	"github.com/gin-gonic/gin"
  )

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
    c.JSON(200, gin.H{
      "message": "create student complete!",
    })
  })

  router.Run() // listens on 0.0.0.0:8080 by default

}