package main

import "github.com/gin-gonic/gin"

func main() {
  router := gin.Default()

  //Get
  router.GET("/student", func(c *gin.Context) {
    c.JSON(200, gin.H{
      "message": "get student complete!",
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