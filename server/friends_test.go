package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"encoding/json"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// 1st version
/*func TestValidateFriendPair(t *testing.T)  {
	err := validateFriendPair("001", "002")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateFriendPair_EmptyStudentID(t *testing.T) {
	err := validateFriendPair("", "002")

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestValidateFriendPair_SameStudent(t *testing.T) {
	err := validateFriendPair("001", "001")

	if err == nil {
		t.Error("expected error, got nil")
	}
}*/

// 2nd version
func TestValidateFriendPair(t *testing.T) {
	tests := []struct {
		name      string
		studentID string
		friendID  string
		wantError bool
	}{
		{"two different students", "0231", "0232", false},
		{"nobody is their own friend", "0231", "0231", true},
		{"missing friend", "0231", "", true},
		{"missing student", "", "0232", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFriendPair(tt.studentID, tt.friendID)

			if (err != nil) != tt.wantError {
				t.Errorf("validateFriendPair() error = %v, wantError %v, case: %s", err, tt.wantError, tt.name)
			}
		})
	}
}

// Every student this test creates starts with test_, so we can find and remove
// our own rows afterwards without touching anybody else's records.
const testPrefix = "test_"

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", databaseURL())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		t.Skip("database is not running; start it with: docker compose up -d --wait")
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM friendship WHERE student_id LIKE $1 OR friend_id LIKE $1", testPrefix+"%")
		db.Exec("DELETE FROM student WHERE student_id LIKE $1", testPrefix+"%")
		db.Close()
	})
	return db
}

func newTestRouter(store *Store) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/friends", handleListFriends(store))
	router.POST("/friend", handleAddFriend(store))
	return router
}

func call(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestAddFriendIsVisibleFromBothSides(t *testing.T) {
	db := openTestDatabase(t)
	store := NewStore(db)
	router := newTestRouter(store)

	mali := testPrefix + "mali"
	arun := testPrefix + "arun"

	for id, name := range map[string]string{mali: "Test Mali", arun: "Test Arun"} {
		if _, err := store.CreateStudent(t.Context(), id, name, id+"@example.com"); err != nil {
			t.Fatal(err)
		}
	}

	response := call(router, http.MethodPost, "/friend", `{"studentId":"`+mali+`","friendId":"`+arun+`"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("adding a friend: expected 201, got %d: %s", response.Code, response.Body)
	}

	for _, pair := range [][2]string{{mali, arun}, {arun, mali}} {
		response = call(router, http.MethodGet, "/friends?studentId="+pair[0], "")
		if response.Code != http.StatusOK {
			t.Fatalf("listing friends of %s: expected 200, got %d", pair[0], response.Code)
		}
		var friends []FriendResponse
		if err := json.Unmarshal(response.Body.Bytes(), &friends); err != nil {
			t.Fatal(err)
		}
		if len(friends) != 1 || friends[0].StudentID != pair[1] {
			t.Fatalf("friends of %s: expected only %s, got %+v", pair[0], pair[1], friends)
		}
		if friends[0].Since == nil {
			t.Errorf("friends of %s: expected a start date on a new friendship", pair[0])
		}
	}
}

func TestFriendsOfAStudentWithNoFriends(t *testing.T) {
	db := openTestDatabase(t)
	store := NewStore(db)
	router := newTestRouter(store)

	lonely := testPrefix + "lonely"
	if _, err := store.CreateStudent(t.Context(), lonely, "Test lonely", "lonely@example.com"); err != nil {
		t.Fatal(err)
	}

	response := call(router, http.MethodGet, "/friends?studentId="+lonely, "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 for a student with no friends, got %d", response.Code)
	}
	if body := strings.TrimSpace(response.Body.String()); body != "[]" {
		t.Errorf("expected an empty list, got %s", body)
	}
}

func TestFriendsOfAStudentWhoDoesNotExist(t *testing.T) {
	db := openTestDatabase(t)
	store := NewStore(db)
	router := newTestRouter(store)

	nobody := testPrefix + "nobody"

	response := call(router, http.MethodGet, "/friends?studentId="+nobody, "")
	if response.Code != http.StatusNotFound {
		t.Errorf("expected 404 for a student who does not exist, got %d", response.Code)
	}
}