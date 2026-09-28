package main

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"os"
)

func databaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://student_user:student_password@127.0.0.1:5434/student_register?sslmode=disable&connect_timeout=5"
}

func main() {
	//Connect DB
	db, err := sql.Open("pgx", "postgres://student_user:student_password@127.0.0.1:5434/student_register?sslmode=disable&connect_timeout=5")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Could not connect to the database: ", err)
	}

	fmt.Println("Connected to the student database")

	store := NewStore(db)

	router := gin.Default()
	router.StaticFile("/", "../web-app/index.html")

	router.GET("/student", handleFindStudent(store))
	router.POST("/student", handleCreateStudent(store))
	router.GET("/friends", handleListFriends(store))
	router.POST("/friend", handleAddFriend(store))

	if err := router.Run("127.0.0.1:8080"); err != nil {
		log.Println("Could not start API:", err)
	}

	router.Run() // listens on 0.0.0.0:8080 by default
}
