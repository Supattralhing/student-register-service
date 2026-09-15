# 11. Show and add friends

**Today's result:** `GET /friends?studentId=0232` lists Arun's friends, and `POST /friend` makes two students friends with each other.

This is the longest lesson of the week. Split it into two sittings:

* **Sitting A** — registration creates a friends list, and Go can show a student's friends.
* **Sitting B** — Go can add a friend.

Keep the database running throughout.

---

## Sitting A

### 1. Go back into code you already finished

Lesson 10 gave every **existing** student a friends list. But `POST /student` still knows nothing about lists, so the next student registered through Postman will have no list at all, and lesson 10's backfill has already been run.

So today you edit lesson 9's registration code.

This feels wrong the first time. It is not. Almost every new feature means going back into code that was finished and correct, and making it do one more thing. It does not mean lesson 9 was done badly.

### 2. Two writes that must both happen

Registering a student now means writing to **two** tables: a row in `student` and a row in `friend_group`.

**Predict first.** Suppose Go saves the student, and then the computer loses power before it saves the friends list. What is in the database when it comes back on? Is that a problem?

That situation — half the work saved — is what a **transaction** prevents. A transaction says: *treat these statements as one. Either all of them happen, or none of them do.*

| Code | Meaning |
| --- | --- |
| `db.BeginTx(ctx, nil)` | Start a transaction. Work done on it is not final yet. |
| `transaction.ExecContext(...)` | Run a statement as part of this transaction. |
| `transaction.Commit()` | Make everything in the transaction final, together. |
| `transaction.Rollback()` | Undo everything in the transaction. |
| `defer transaction.Rollback()` | Arrange to undo it if we leave this function without committing. |

`Rollback` after a successful `Commit` does nothing, so `defer` is safe to write once at the top. That is the usual Go shape: say how to clean up immediately, before the things that might fail.

### 3. Change POST /student

In your `POST /student` handler, replace the single `INSERT` with a transaction. The new group's name is the student's own ID with `g-` in front, matching the backfill from lesson 10.

```go
transaction, err := db.BeginTx(c.Request.Context(), nil)
if err != nil {
    log.Println("Could not start saving:", err)
    c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save student"})
    return
}
defer transaction.Rollback()

groupID := "g-" + request.StudentID

var student StudentResponse
err = transaction.QueryRowContext(c.Request.Context(),
    `INSERT INTO student (student_id, name, email, friend_group_id)
     VALUES ($1, $2, $3, $4)
     RETURNING student_id, name, email`,
    request.StudentID, request.Name, request.Email, groupID,
).Scan(&student.StudentID, &student.Name, &student.Email)
```

Keep your existing duplicate-ID check (`23505`) after that `Scan`. Then add the second write and the commit:

```go
if _, err := transaction.ExecContext(c.Request.Context(),
    "INSERT INTO friend_group (group_id) VALUES ($1)", groupID); err != nil {
    log.Println("Could not create the friends list:", err)
    c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save student"})
    return
}

if err := transaction.Commit(); err != nil {
    log.Println("Could not finish saving:", err)
    c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save student"})
    return
}
c.JSON(http.StatusCreated, student)
```

Wait — the `student` row points at a `friend_group` row that is written *afterwards*. The foreign key from lesson 10 should refuse that. It does not, because both statements are inside one transaction: Postgres checks the promise when the transaction commits, not between statements. Both rows exist by then.

Run `go run .` from **server**, register a student in Postman, then check in DBeaver:

```sql
SELECT student_id, friend_group_id FROM student ORDER BY student_id;
SELECT * FROM friend_group ORDER BY group_id;
```

**Checkpoint:** the new student has a `friend_group_id`, and a matching empty row exists in `friend_group`.

### 4. Ask the database for a student's friends

Try the question in DBeaver first, before writing any Go. Give Mali a friend by hand so there is something to find:

```sql
UPDATE friend_group SET friends = '{0232}' WHERE group_id = 'g-0231';
```

Now read it back:

```sql
SELECT friend.student_id, friend.name, friend.email
FROM student me
JOIN friend_group list ON list.group_id = me.friend_group_id
JOIN student friend ON friend.student_id = ANY (list.friends)
WHERE me.student_id = '0231'
ORDER BY friend.name;
```

Read it as three steps:

| Line | In everyday words |
| --- | --- |
| `FROM student me` | Start from one student — call them `me`. |
| `JOIN friend_group list ON list.group_id = me.friend_group_id` | Fetch my friends list. |
| `JOIN student friend ON friend.student_id = ANY (list.friends)` | For every ID on that list, fetch that student's record. |
| `WHERE me.student_id = '0231'` | And `me` is Mali. |

`ANY (list.friends)` means *matches any value in this array*. The same table, `student`, is used twice with two different names, because the question mentions two different people: me, and my friend.

Expected: one row, Arun.

### 5. Add GET /friends

Add this route next to your existing ones:

```go
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
        `SELECT friend.student_id, friend.name, friend.email
         FROM student me
         JOIN friend_group list ON list.group_id = me.friend_group_id
         JOIN student friend ON friend.student_id = ANY (list.friends)
         WHERE me.student_id = $1
         ORDER BY friend.name`,
        studentID,
    )
    if err != nil {
        log.Println("Could not read friends:", err)
        c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not read friends"})
        return
    }
    defer rows.Close()

    friends := []StudentResponse{}
    for rows.Next() {
        var friend StudentResponse
        if err := rows.Scan(&friend.StudentID, &friend.Name, &friend.Email); err != nil {
            log.Println("Could not read friends:", err)
            c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not read friends"})
            return
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
```

And the small helper it uses, above `main`:

```go
func studentExists(ctx context.Context, db *sql.DB, studentID string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM student WHERE student_id = $1)",
		studentID,
	).Scan(&exists)
	return exists, err
}
```

Add `"context"` to your imports.

| Code | Meaning |
| --- | --- |
| `db.QueryContext` | Ask for **many** rows. `QueryRowContext`, from lesson 8, asks for one. |
| `rows.Next()` | Move to the next row; `false` when there are no more. |
| `defer rows.Close()` | Release the rows when we are finished with them. |
| `rows.Err()` | Ask whether the loop stopped because of a problem rather than because it ran out of rows. |
| `friends := []StudentResponse{}` | Start with an **empty list**, not `var friends []StudentResponse`. |
| `append` | Add one item to the end of a list. |

That `friends := []StudentResponse{}` line matters more than it looks. An unset list in Go turns into `null` in JSON; an empty one turns into `[]`. The page expects a list it can count, so we always send a list.

### 6. Three answers that are not the same

Try all three in Postman:

| Request | Expected | Why |
| --- | --- | --- |
| `GET /friends?studentId=0231` | `200`, a list with Arun | Mali has one friend. |
| `GET /friends?studentId=0235` | `200`, `[]` | Nida is a real student who has no friends yet. |
| `GET /friends?studentId=9999` | `404` | There is no such student to ask about. |

**Checkpoint:** in your own words, why is "this student has no friends" a success and "there is no such student" a failure? A front desk clerk would answer these two questions with different sentences. So does the API.

---

## Sitting B

### 7. Making two people friends changes two lists

Mali's list must gain Arun, and Arun's list must gain Mali. **Predict first:** if only the first write succeeded, what would Mali see? What would Arun see? Which one of them would call the office to complain?

You already have the tool for this from sitting A: a transaction.

### 8. The database cannot help us here

In lesson 10 the foreign key stopped you putting a made-up list name into `student.friend_group_id`. Try the same trick inside an array:

```sql
UPDATE friend_group SET friends = '{9999}' WHERE group_id = 'g-0235';
```

Expected: **it works.** There is no student `9999`. Postgres cannot put a foreign key on a value inside an array, so it has no way to object.

Undo it before going on:

```sql
UPDATE friend_group SET friends = '{}' WHERE group_id = 'g-0235';
```

So our Go code has to do the checking that the database cannot. Remember this moment; lesson 12 comes back to it.

### 9. Add POST /friend

A new request type, next to your other `struct`s:

```go
type AddFriendRequest struct {
	StudentID string `json:"studentId" binding:"required"`
	FriendID  string `json:"friendId" binding:"required"`
}
```

And the route:

```go
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

    const addToOneList = `UPDATE friend_group
                          SET friends = array_append(friends, $2)
                          WHERE group_id = (SELECT friend_group_id FROM student WHERE student_id = $1)
                            AND NOT ($2 = ANY (friends))`

    if _, err := transaction.ExecContext(c.Request.Context(), addToOneList, studentID, friendID); err != nil {
        log.Println("Could not save the friendship:", err)
        c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "Could not save the friendship"})
        return
    }
    if _, err := transaction.ExecContext(c.Request.Context(), addToOneList, friendID, studentID); err != nil {
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
```

| Code | Meaning |
| --- | --- |
| `array_append(friends, $2)` | Make a new list that is the old one with this ID added at the end. |
| `NOT ($2 = ANY (friends))` | Only if that ID is not on the list already, so clicking twice changes nothing. |
| `const addToOneList` | The same statement is used twice with the two IDs swapped, so it is written once. |
| `c.Status(http.StatusCreated)` | Answer `201` with no body: there is no new record to show back. |

The same statement, run twice with the arguments the other way round, is what makes friendship go both ways.

### 10. Prove the three-friend story

In Postman:

1. `POST /friend` with `{"studentId":"0231","friendId":"0232"}` — Mali and Arun.
2. `POST /friend` with `{"studentId":"0232","friendId":"0234"}` — Arun and Somchai.

Then:

| Request | Expected |
| --- | --- |
| `GET /friends?studentId=0231` | Arun only |
| `GET /friends?studentId=0232` | Mali **and** Somchai |
| `GET /friends?studentId=0234` | Arun only |

Mali and Somchai have never met, and the database says so. Also try `POST /friend` with `{"studentId":"0231","friendId":"9999"}` and expect `404`.

### 11. Turn on the page

Copy the supplied page over the old one. From the project's top folder:

```sh
cp docs/week02/handouts/index-lesson-11.html web-app/index.html
```

Your teacher wrote this page. You did not, and you will not: your job is the API behind it. This is normal. Most backend engineers spend their whole career making pages work that somebody else wrote.

Restart Go from **server**, open **http://127.0.0.1:8080/**, and look up `0232`. Arun's record appears with his friends underneath, and a small form to add another.

**Checkpoint:** a teammate asks, *"why isn't `friends` just a column on `student`? Why is there a second table at all?"* What would you answer? It is a fair question, and you may not have a good answer. Keep it in mind for lesson 12.

Next: [12. "When did they become friends?"](12-when-did-they-become-friends.md).
