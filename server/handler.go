package main

import (
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"net/http"
	"strings"
)

type HandlerFunc struct {
	store *Store
}

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

func cleanStudentID(raw string) string {
	return strings.TrimSpace(raw)
}

func validateFriendPair(studentID, friendID string) error {
	if studentID == "" || friendID == "" {
		return errors.New("studentId and friendId are required")
	}
	if studentID == friendID {
		return errors.New("a student cannot be their own friend")
	}
	return nil
}

// Get Student
func handleFindStudent(store *Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		studentID := cleanStudentID(c.Query("studentId"))
		if studentID == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "studentId is required"})
			return
		}

		student, err := store.FindStudent(c.Request.Context(), studentID)
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
	}
}

// Post Student
func handleCreateStudent(store *Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request CreateStudentRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Send studentId, name, and email as nonempty JSON text fields"})
			return
		}
		studentID := cleanStudentID(request.StudentID)
		name := cleanStudentID(request.Name)
		email := cleanStudentID(request.Email)
		if studentID == "" || name == "" || email == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "studentId, name, and email are required"})
			return
		}

		student, err := store.CreateStudent(c.Request.Context(), studentID, name, email)
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
		c.JSON(http.StatusCreated, student)
	}
}

// Get Friend
func handleListFriends(store *Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		studentID := cleanStudentID(c.Query("studentId"))
		if studentID == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "studentId is required"})
			return
		}

		exists, err := store.studentExists(c.Request.Context(), studentID)
		if err != nil {
			log.Println("Could not check student:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not read friends"})
			return
		}
		if !exists {
			c.JSON(http.StatusNotFound, ErrorResponse{Message: "Student not found"})
			return
		}

		friends, err := store.ListFriends(c.Request.Context(), studentID)
		if err != nil {
			log.Println("Could not read friends:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not read friends"})
			return
		}
		c.JSON(http.StatusOK, friends)
	}
}

// Post Friend
func handleAddFriend(store *Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request AddFriendRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Send studentId and friendId as nonempty JSON text fields"})
			return
		}
		studentID := cleanStudentID(request.StudentID)
		friendID := cleanStudentID(request.FriendID)

		if err := validateFriendPair(studentID, friendID); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
			return
		}

		for _, id := range []string{studentID, friendID} {
			exists, err := store.studentExists(c.Request.Context(), id)
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

		err := store.AddFriend(c.Request.Context(), studentID, friendID)
		if err != nil {
			log.Println("Could not save the friendship:", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save the friendship"})
			return
		}
		c.Status(http.StatusCreated)

	}
}
