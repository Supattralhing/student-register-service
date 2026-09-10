# 9. Make Go save a new student

**Today's result:** POST /student saves a registration, and you can find it afterward with GET and DBeaver.

Think of the request body as a completed registration form. Go checks the form, gives it to PostgreSQL, and returns the saved student details as a receipt. Start with the GET program from lesson 8. PostgreSQL should be running.

## 1. Understand the database instruction first

Read this example together before running anything:

~~~sql
INSERT INTO student (student_id, name, email)
VALUES ('0233', 'Nida Chai', 'nida@example.com')
RETURNING student_id, name, email;
~~~

It means: “Add this student. Then show me the details of the row just added.” RETURNING lets PostgreSQL return values from a changed row. See the [PostgreSQL guide](https://www.postgresql.org/docs/17/dml-returning.html).

Save ID **0233** for the Postman exercise below; do not run this exact example in DBeaver first. Otherwise it will already exist when you POST it.

**Predict:** If the register already contains 0233, should a new registration replace that student? Our rule is **no**: tell the caller the ID is already in use.

## 2. Describe the incoming form

Add this **above func main()** in server/main.go, alongside the response types:

~~~go
type CreateStudentRequest struct {
	StudentID string `json:"studentId" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Email     string `json:"email" binding:"required"`
}
~~~

If you already have a request struct from lesson 5, update it to this shape instead of declaring it twice.

CreateStudentRequest describes what the caller sends. StudentResponse describes what we send back. They happen to contain the same fields today, but they have different jobs. Both GET and a successful POST return the same StudentResponse shape.

The **required** labels ask Gin to reject missing or empty fields. We also remove surrounding spaces and reject values containing only spaces. This lesson checks that all three values are nonempty text; it does not check email format or whether an address exists. Gin's [binding guide](https://gin-gonic.com/en/docs/binding/binding-and-validation/) explains how ShouldBindJSON fills and checks a struct.

## 3. Add one import

Inside the existing import block, add:

~~~go
"github.com/jackc/pgx/v5/pgconn"
~~~

Keep the pgx/stdlib import too. **pgconn** lets us recognize PostgreSQL's message for a duplicate ID.

## 4. Add the POST instruction

Inside main, add this **after the closing }) of the GET handler and before router.Run**. If your earlier code already has POST /student, replace that whole handler with this one. Do not register it twice.

~~~go
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

	var student StudentResponse
	err := db.QueryRowContext(c.Request.Context(),
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

	c.JSON(http.StatusCreated, student)
})
~~~

The steps are: **read the form → check it → save the student → return the saved row**.

ShouldBindJSON fills the request form. The double vertical bar means “or,” so any blank field causes a 400 reply. SQL reserves $1, $2, and $3 for the three supplied values in that order.

Backticks around SQL let the Go text span several lines. This insert uses QueryRowContext because RETURNING produces a row to read with Scan. An insert that returns no row would normally use ExecContext instead.

The pgError block is supplied code for recognizing a database error. PostgreSQL uses code **23505** when a value violates a uniqueness rule; in this table, that means the student ID already exists. We reply with **409 Conflict**. The primary key enforces this rule even if two requests arrive together. See [PostgreSQL's error codes](https://www.postgresql.org/docs/17/errcodes-appendix.html) and [pgx error handling](https://github.com/jackc/pgx/wiki/Error-Handling).

We return success only after the insert succeeds. A single statement here is saved automatically; there is no separate Go commit step in this example. Database problem details go to the terminal; the caller gets a short message.

For an answer sheet with the complete imports and both handlers, open [the finished reference](reference/main.go.txt). Its .txt ending keeps it out of Go's normal build. If needed, copy its contents into server/main.go.

## 5. Restart and send a registration

Save main.go. In the terminal running Go, press **Ctrl+C**. Then, from **server**:

~~~sh
gofmt -w main.go
go run .
~~~

In Postman:

1. Select **POST**.
2. Enter **http://localhost:8080/student**.
3. Choose **Body → raw → JSON**.
4. Check that the request has **Content-Type: application/json**.
5. Paste this form and click **Send** once:

~~~json
{
  "studentId": "0233",
  "name": "Nida Chai",
  "email": "nida@example.com"
}
~~~

Use double quotes in JSON, including around "0233". Expect **201 Created** and:

~~~json
{
  "studentId": "0233",
  "name": "Nida Chai",
  "email": "nida@example.com"
}
~~~

201 means “created.” The reply looks like the form we sent, so we still need independent evidence that it was saved.

## 6. Prove it was saved in two ways

First, send a GET request in Postman:

~~~text
http://localhost:8080/student?studentId=0233
~~~

Expect **200 OK** with Nida's details.

Then, in DBeaver, refresh the table or run:

~~~sql
SELECT student_id, name, email
FROM student
WHERE student_id = '0233';
~~~

You should see the same student. There should now be three rows if you started with the original sample data and added only Nida.

## 7. Try mistakes on purpose

Before each try, predict the outcome. Use a fresh ID for new registrations if you repeat this lesson.

| Try in Postman | Expected result | What to check in DBeaver |
| --- | --- | --- |
| Send the same 0233 POST again. | 409 Conflict: ID already exists. | There is still only one row with 0233. |
| Change the name but POST with existing ID 0233. | 409 Conflict. | Nida's original name is unchanged. POST here adds; it does not update. |
| Use 0234 but remove name. | 400 Bad Request. | No 0234 row was added. |
| Use 0234 with a name containing only spaces. | 400 Bad Request. | No 0234 row was added. |
| Remove a comma between JSON fields. | 400 Bad Request. | Nothing was added. Restore valid JSON afterward. |
| Send studentId as the JSON number 234. | 400 Bad Request; our field expects text. | Nothing was added. Use a quoted ID. |

The malformed-form cases return a message explaining that the three JSON text fields are needed. A database failure returns **500 Internal Server Error**, meaning the API could not finish the operation; check the Go terminal for details.

Optional teacher demonstration: keep Go running, run `docker compose stop` from the project's top folder, and send a GET request. Expect a 500 database error, not “Student not found.” Restart PostgreSQL with `docker compose up -d --wait`, then retry. If Go starts while the database is stopped, its startup check fails and the API does not start at all.

## 8. Prove the record outlives the Go program

1. Stop Go with **Ctrl+C**.
2. Start it again with `go run .` from **server**.
3. GET student 0233 again. Nida should still be there.
4. From the **project's top folder**, stop and restart PostgreSQL with `docker compose stop` followed by `docker compose up -d --wait`.
5. GET 0233 again and refresh DBeaver. The saved volume keeps the record through this restart too.

**Ready to finish:** Register another fictional student using an unused ID. Find the student in both apps. Explain which part reads the form, which part saves it, and why it survives restarting Go.

Teacher's final prompt: “A colleague says the POST response proves the student was saved. What extra check would you show them?”
