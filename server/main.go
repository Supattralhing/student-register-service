package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"net/http"
	"strings"
	"time"
)

type StudentResponse struct {
	StudentID string `json:"studentId"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

type FriendResponse struct {
	StudentID string  `json:"studentId"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Since     *string `json:"since"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type CreateStudentRequest struct {
	StudentID string `json:"studentId" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Email     string `json:"email" binding:"required"`
}

type AddFriendRequest struct {
	StudentID string `json:"studentId" binding:"required"`
	FriendID  string `json:"friendId" binding:"required"`
}

func studentExists(ctx context.Context, db *sql.DB, studentID string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM student WHERE student_id = $1)",
		studentID,
	).Scan(&exists)
	return exists, err
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

	router := gin.Default()
	router.StaticFile("/", "../web-app/index.html")

	//Get Student
	router.GET("/student", func(c *gin.Context) {
		studentID := strings.TrimSpace(c.Query("studentId"))
		if studentID == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "studentId is required"})
			return
		}

		var student StudentResponse
		err := db.QueryRowContext(c.Request.Context(),
			"SELECT student_id, name, email FROM student WHERE student_id = $1",
			studentID,
		).Scan(&student.StudentID, &student.Name, &student.Email)

		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, ErrorResponse{Message: "Student not found"})
			return
		}
		if err != nil {
			log.Println("Could not read student:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not read student"})
			return
		}
		c.JSON(http.StatusOK, student)
	})

	//GET Friend
	router.GET("/friends", func(c *gin.Context) {
		studentID := strings.TrimSpace(c.Query("studentId"))
		if studentID == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "studentId is required"})
			return
		}

		exists, err := studentExists(c.Request.Context(), db, studentID)
		if err != nil {
			log.Println("Could not check student:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not read friends"})
			return
		}
		if !exists {
			c.JSON(http.StatusNotFound, ErrorResponse{Message: "Student not found"})
			return
		}

		rows, err := db.QueryContext(c.Request.Context(),
			`SELECT s.student_id, s.name, s.email, f.since
     		FROM friendship f
     		JOIN student s ON s.student_id = f.friend_id
    		WHERE f.student_id = $1
     		ORDER BY s.name`,
			studentID,
		)
		if err != nil {
			log.Println("Could not read friends:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not read friends"})
			return
		}
		defer rows.Close()

		friends := []FriendResponse{}
		for rows.Next() {
			var friend FriendResponse
			var since sql.NullTime
			if err := rows.Scan(&friend.StudentID, &friend.Name, &friend.Email, &since); err != nil {
				log.Println("Could not read friends:", err)
				c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not read friends"})
				return
			}
			if since.Valid {
				day := since.Time.Format("2006-01-02")
				friend.Since = &day
			}
			friends = append(friends, friend)
		}
		if err := rows.Err(); err != nil {
			log.Println("Could not read friends:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not read friends"})
			return
		}
		c.JSON(http.StatusOK, friends)
	})

	//Post
	router.POST("/student", func(c *gin.Context) {
		var request CreateStudentRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Send studentId, name, and email as nonempty JSON text fields"})
			return
		}

		request.StudentID = strings.TrimSpace(request.StudentID)
		request.Name = strings.TrimSpace(request.Name)
		request.Email = strings.TrimSpace(request.Email)
		if request.StudentID == "" || request.Name == "" || request.Email == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "studentId, name, and email are required"})
			return
		}

		transaction, err := db.BeginTx(c.Request.Context(), nil)
		if err != nil {
			log.Println("Could not start saving:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save student"})
			return
		}
		defer transaction.Rollback()

		var student StudentResponse
		err = transaction.QueryRowContext(c.Request.Context(),
			`INSERT INTO student (student_id, name, email)
     		VALUES ($1, $2, $3)
     		RETURNING student_id, name, email`,
			request.StudentID, request.Name, request.Email,
		).Scan(&student.StudentID, &student.Name, &student.Email)

		if err != nil {
			var pgError *pgconn.PgError
			if errors.As(err, &pgError) && pgError.Code == "23505" {
				c.JSON(http.StatusConflict, ErrorResponse{Message: "A student with this ID already exists"})
				return
			}
			log.Println("Could not save student:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save student"})
			return
		}

		if err := transaction.Commit(); err != nil {
			log.Println("Could not finish saving:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save student"})
			return
		}

		c.JSON(http.StatusCreated, student)
	})
	//POST /friend
	router.POST("/friend", func(c *gin.Context) {
		var request AddFriendRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Send studentId and friendId as nonempty JSON text fields"})
			return
		}

		studentID := strings.TrimSpace(request.StudentID)
		friendID := strings.TrimSpace(request.FriendID)
		if studentID == "" || friendID == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "studentId and friendId are required"})
			return
		}
		if studentID == friendID {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "A student cannot be their own friend"})
			return
		}

		for _, id := range []string{studentID, friendID} {
			exists, err := studentExists(c.Request.Context(), db, id)
			if err != nil {
				log.Println("Could not check student:", err)
				c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save the friendship"})
				return
			}
			if !exists {
				c.JSON(http.StatusNotFound, ErrorResponse{Message: "Student " + id + " is not registered"})
				return
			}
		}

		transaction, err := db.BeginTx(c.Request.Context(), nil)
		if err != nil {
			log.Println("Could not start saving:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save the friendship"})
			return
		}
		defer transaction.Rollback()

		today := time.Now().Format("2006-01-02")
		const addOneDirection = `INSERT INTO friendship (student_id, friend_id, since)
                         VALUES ($1, $2, $3)
                         ON CONFLICT (student_id, friend_id) DO NOTHING`

		if _, err := transaction.ExecContext(c.Request.Context(), addOneDirection, studentID, friendID, today); err != nil {
			log.Println("Could not save the friendship:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save the friendship"})
			return
		}
		if _, err := transaction.ExecContext(c.Request.Context(), addOneDirection, friendID, studentID, today); err != nil {
			log.Println("Could not save the friendship:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save the friendship"})
			return
		}

		if err := transaction.Commit(); err != nil {
			log.Println("Could not finish saving:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save the friendship"})
			return
		}
		c.Status(http.StatusCreated)
	})

	if err := router.Run("127.0.0.1:8080"); err != nil {
		log.Println("Could not start API:", err)
	}

	router.Run() // listens on 0.0.0.0:8080 by default
}
