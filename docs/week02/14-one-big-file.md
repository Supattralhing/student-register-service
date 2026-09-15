# 14. One big file is hard to read

**Today's result:** the same working API, spread over three files that each do one job — and the database password is no longer written in the code.

Nothing about what the API *does* changes today. Every request still gives the same answer. This is called **refactoring**: changing the shape of code without changing its behaviour.

## 1. Look at what you have

Open `server/main.go` and scroll from top to bottom without reading.

It is around 250 lines. `main` starts by connecting to a database, then defines four handlers, each of which contains SQL, HTTP status codes, error messages and JSON, all mixed together. To find the friends query you have to scroll past registration.

**Predict first:** if a teammate asked you to change every SQL query to also fetch a phone number, how many places would you have to look, and how would you be sure you found them all?

## 2. Find the seams

Read through the file and sort every line into one of three jobs:

| Job | Answers the question | Examples |
| --- | --- | --- |
| **Start up** | How does this program begin? | Connecting to the database, listing the routes, starting the server. |
| **Handle a request** | What does a request mean, and what do we answer? | Reading `studentId`, checking it is not empty, choosing `200`, `404` or `400`, shaping the JSON. |
| **Talk to the database** | How do we read or write a record? | Every line of SQL, `Scan`, transactions. |

Those three jobs are the three files:

```text
server/
├── main.go      start up
├── handler.go   handle a request
└── store.go     talk to the database
```

Every file still begins with `package main`, and Go treats them as one program. You do not have to tell Go about the new files; it reads every `.go` file in the folder. Nothing is imported between them.

## 3. Move the database work into store.go

Create `server/store.go`. It holds one new type, and every piece of SQL you have written.

```go
package main

import (
	"context"
	"database/sql"
	"time"
)

// Store is the only part of the program that writes SQL. Everything else asks
// Store for what it needs.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}
```

| Code | Meaning |
| --- | --- |
| `type Store struct { db *sql.DB }` | A thing that keeps a database connection. |
| `func NewStore(db *sql.DB) *Store` | Make one, given a connection. |
| `func (s *Store) FindStudent(...)` | A **method**: a function that belongs to `Store`, so it can use `s.db`. |

Now move the queries across, one per job. The full file is in [reference/lesson-15/store.go.txt](reference/lesson-15/store.go.txt) — move them yourself first, then compare.

```go
func (s *Store) FindStudent(ctx context.Context, studentID string) (StudentResponse, error) {
	var student StudentResponse
	err := s.db.QueryRowContext(ctx,
		"SELECT student_id, name, email FROM student WHERE student_id = $1",
		studentID,
	).Scan(&student.StudentID, &student.Name, &student.Email)
	return student, err
}
```

Notice what `FindStudent` does **not** do. It does not choose a status code. It does not write a message for a person to read. It hands back the student and any problem, and lets the caller decide what that means. Deciding is the handler's job.

By the end you should have: `FindStudent`, `CreateStudent`, `StudentExists`, `ListFriends`, `AddFriend`.

## 4. Move the request work into handler.go

Create `server/handler.go`. It holds every `struct` with JSON tags, and the four handlers.

The handlers each need a `Store`. The shape for that is a function that takes the store and gives back a handler:

```go
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
```

Read the outer function out loud: *"give me a `Store`, and I will give you back a handler that uses it."* The handler is the same code you already had; it now calls `store.FindStudent` instead of writing SQL itself.

While moving `POST /friend`, pull its two rules out into a small function of their own:

```go
// validateFriendPair holds the two rules about a friendship that do not need a
// database to check. Keeping them in their own function makes them easy to test.
func validateFriendPair(studentID, friendID string) error {
	if studentID == "" || friendID == "" {
		return errors.New("studentId and friendId are required")
	}
	if studentID == friendID {
		return errors.New("a student cannot be their own friend")
	}
	return nil
}
```

This function has no database, no HTTP, no Gin. Give it two pieces of text and it gives back an answer. Remember that; lesson 15 comes straight back to it.

Do the same for the trimming:

```go
func cleanStudentID(raw string) string {
	return strings.TrimSpace(raw)
}
```

## 5. What is left in main.go

```go
func main() {
	db, err := sql.Open("pgx", databaseURL())
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
	router.POST("/student", handleRegisterStudent(store))
	router.GET("/friends", handleListFriends(store))
	router.POST("/friend", handleAddFriend(store))

	if err := router.Run("127.0.0.1:8080"); err != nil {
		log.Println("Could not start API:", err)
	}
}
```

Around thirty lines, and the four routes are visible at a glance. A new teammate can read this and know what the API does.

## 6. Take the password out of the code

One more small change. This line has been in your code since lesson 8:

```go
"postgres://student_user:student_password@127.0.0.1:5434/student_register?sslmode=disable&connect_timeout=5"
```

The password is written into a file that goes into Git and gets shared. Ours is a practice password for a database on your own computer, so nothing is at risk here. The habit still matters, because the same line in a real project would send a real password to everyone who ever reads the code.

Add this to `main.go`:

```go
// The database address lives outside the code now, so the password is not
// written into a file we share. If DATABASE_URL is not set, we use the
// practice database from lesson 6.
func databaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://student_user:student_password@127.0.0.1:5434/student_register?sslmode=disable&connect_timeout=5"
}
```

Add `"os"` to your imports. An **environment variable** is a setting given to a program when it starts, from outside the program. Try it:

```sh
DATABASE_URL="postgres://wrong:wrong@127.0.0.1:5434/student_register?sslmode=disable" go run .
```

Expected: the connection fails, which proves the setting was used. Run `go run .` again with nothing in front and it works as before.

## 7. Check nothing changed

```sh
cd server
go build ./...
go run .
```

Run all five checks from lesson 13 step 7 again. Every answer must be identical to before you started. If anything differs, something moved that should not have.

**Checkpoint:** answer the question from step 1 again. Where would the phone number change go now, and how many files would you have to open?

Next: [15. Prove it still works tomorrow](15-prove-it-still-works.md).
