# 8. Make Go find a saved student

**Today's result:** GET /student?studentId=0231 returns Mali's details from PostgreSQL.

In lesson 7, you asked the database a question in DBeaver. Now Go will ask that same question when Postman sends a request. Split this lesson into two sessions if needed: first make the connection, then add the lookup. Keep PostgreSQL running throughout.

## 1. Add the database helper

Open a terminal in **server**, the folder containing go.mod. From the project's top folder:

~~~sh
cd server
go version
go get github.com/gin-gonic/gin@v1.12.0
go get github.com/jackc/pgx/v5/stdlib@v5.7.6
~~~

`go get` downloads reusable Go code and records it in go.mod and go.sum. Gin handles HTTP requests. **pgx** is the driver that lets Go talk to PostgreSQL. These versions keep the examples consistent; the project's Go 1.26.7 setting is sufficient for them.

Go provides a toolbox called `database/sql`; pgx supplies its PostgreSQL connection. See the [pgx driver documentation](https://pkg.go.dev/github.com/jackc/pgx/v5/stdlib).

## 2. First, prove that Go can connect

Save a copy of your earlier main.go outside the server folder, or in a Git commit, before editing it. Do not leave a second .go file containing another main function in the same folder.

For this short checkpoint, replace **server/main.go** with:

~~~go
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, err := sql.Open("pgx", "postgres://student_user:student_password@127.0.0.1:5434/student_register?sslmode=disable&connect_timeout=5")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Could not connect to the database: ", err)
	}

	fmt.Println("Connected to the student database")
}
~~~

Run `go run .` from **server**. Expected: **Connected to the student database**, then the program finishes. There is no HTTP server at this checkpoint, so Postman cannot call it yet.

| Code | Meaning |
| --- | --- |
| The long postgres:// text | The same address, database, and login you gave DBeaver, written on one line. |
| `sslmode=disable` | Use an unencrypted connection for this local Docker exercise. |
| `connect_timeout=5` | Give the connection attempt up to five seconds. |
| `sql.Open` | Prepare a database handle named db, which Go can reuse for database work. |
| `db.Ping()` | Actually contact the database and check that the connection works. |
| `err` | The problem reported by an operation, if there was one. |
| `err != nil` | “If a problem occurred…”; nil means there is no error here. |
| `defer db.Close()` | Arrange to release database resources when main returns normally. |
| `_` before the pgx import | Register the PostgreSQL driver even though we call it through database/sql. |

Open alone does not prove the database is reachable; that is why we use Ping. We create db once and reuse it across requests. See Go's [connection guide](https://go.dev/doc/database/open-handle).

**Checkpoint:** Explain how this connection text matches the DBeaver connection card. Continue when the connection message appears.

## 3. Describe the reply we want to send

A **struct** is a named form with fields. The response struct is the form Go fills in before returning JSON to Postman.

Add these types **above func main()**, below the imports:

~~~go
type StudentResponse struct {
	StudentID string `json:"studentId"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
~~~

We use one response shape for a student and another for a problem message. **string** means text. The json label tells Gin which field name to use in the reply.

| In PostgreSQL | In Go | In the JSON reply |
| --- | --- | --- |
| student_id | StudentID | studentId |
| name | Name | name |
| email | Email | email |

## 4. Add the GET instruction

Replace the **whole import block** with this one. Go requires us to list the tools this file actually uses:

~~~go
import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)
~~~

Inside main, add the following **after the connection-success Println line and before the closing brace of main**:

~~~go
router := gin.Default()
router.StaticFile("/", "../web-app/index.html")

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

if err := router.Run("127.0.0.1:8080"); err != nil {
	log.Println("Could not start API:", err)
}
~~~

If applying this to your own completed API, replace its existing GET handler instead of registering a second GET /student. Keep only one router and one router.Run call. Your existing POST handler can stay until lesson 9 replaces it. The standalone example built on this page has only GET so far.

`router.StaticFile` also lets Go send the registration page to your browser at `http://127.0.0.1:8080/`. Start Go from **server** so it can find `../web-app/index.html`. The page's **Find a student** form uses the GET instruction above; its registration form will work after lesson 9 adds POST.

Run these in **server**:

~~~sh
gofmt -w main.go
go run .
~~~

gofmt tidies the layout. This time the program keeps running, waiting for requests. Leave the terminal open. After later code changes, save the file, press **Ctrl+C** in this terminal to stop the old program, and run `go run .` again.

## 5. Follow one request through the code

1. Take studentId from the URL. TrimSpace removes spaces at its edges.
2. If it is empty, reply “studentId is required” and stop this request with return.
3. Ask PostgreSQL for the matching student's three fields.
4. Copy the returned values into student with Scan.
5. If there is no match, reply “Student not found.” If the database operation failed, report a server problem.
6. Otherwise, send the filled-in student form as JSON.

In DBeaver you used `WHERE student_id = '0231'`. In Go, **$1** is a space reserved for a value, and the next argument, studentID, supplies it. Keep SQL and input values separate this way; do not build SQL by joining text from the request. The driver treats the supplied ID as data. Go's [query guide](https://go.dev/doc/database/querying) covers parameters, QueryRow, Scan, and ErrNoRows.

QueryRowContext means we expect at most one row. The request's Context lets the database work follow the request's cancellation. For today, the teacher can supply that part.

Scan copies values **in order**: ID into StudentID, name into Name, and email into Email. The **&** means “here is where to put this value.” Scan does not match SQL column names to JSON labels for you.

Names like http.StatusOK are readable Go names for HTTP status numbers. A status number is a short label describing the outcome.

## 6. Check the result in Postman

Select **GET**, enter this URL, and click **Send**. There is no request body:

~~~text
http://localhost:8080/student?studentId=0231
~~~

Expect **200 OK** and:

~~~json
{
  "studentId": "0231",
  "name": "Mali Srisuk",
  "email": "mali@example.com"
}
~~~

The ID is JSON text with quotes. Keep its first zero. If localhost does not connect on your machine, use 127.0.0.1 in the URL.

| URL ending | Expected status | Expected reply |
| --- | --- | --- |
| /student?studentId=0232 | 200 OK | Arun's saved details. |
| /student?studentId=9999 | 404 Not Found | {"message":"Student not found"} |
| /student | 400 Bad Request | {"message":"studentId is required"} |

## 7. Prove that the answer comes from PostgreSQL

In DBeaver, change Mali's name to **Mali Srisuk (updated)** and save/commit it as in lesson 7. Send the same GET request again **without editing or restarting Go**. The new name should appear. Restore the original name afterward.

If you still see the old name, check that the DBeaver change was saved and committed and that both apps use the connection card from lesson 6.

If Postman cannot connect at all, check that go run is still running. If the API returns 500, read the Go terminal's database error and check `docker compose ps` from the project's top folder. A missing student produces 404; a failed database operation produces 500.

**Ready to continue:** Trace ID 0231 from the URL, into the SQL value, into the response struct, and back to Postman. Explain the difference between an absent ID, an unknown ID, and a database problem.

Next: [9. Register a new student](09-register-a-student.md).
