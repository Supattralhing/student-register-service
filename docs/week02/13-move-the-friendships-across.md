# 13. Move the friendships across

**Today's result:** every friendship lives in the `friendship` table, the API shows a start date, and the old design is gone.

The new table exists but is empty. The real friendships are still in the arrays. Today you move them, point Go at the new table, and delete the old one — in that order, checking at each step.

## 1. Move the data, not just the design

**Predict first.** Mali's list says `{0232}`. Arun's says `{0231, 0234}`. How many rows should `friendship` have when the move is finished? Count them before running anything.

The tool for the move is `unnest`. It turns one array into one row per element:

```sql
SELECT unnest(ARRAY['0231', '0232', '0234']);
```

Expected: three rows. This is the whole idea of the migration in one line — *the values were sitting inside a column; now each one is a row.*

## 2. Look for ghosts first

Run statement 2 from [003-friendship.sql](../../database/migrations/003-friendship.sql):

```sql
SELECT s.student_id AS listed_by, listed.friend_id AS ghost_id
FROM student s
JOIN friend_group g ON g.group_id = s.friend_group_id
CROSS JOIN LATERAL unnest(g.friends) AS listed(friend_id)
WHERE listed.friend_id NOT IN (SELECT student_id FROM student);
```

Expected: one row — the ghost `9999` you left in Nida's list in lesson 12.

`CROSS JOIN LATERAL unnest(...)` is the same idea as before: for each student, spread their friends array out into rows so we can look at each ID on its own.

**Predict first:** what will happen if you try to copy that ghost into `friendship`, where both columns are foreign keys?

Try it. Run statement 4 now, before cleaning anything:

```sql
INSERT INTO friendship (student_id, friend_id, since)
SELECT s.student_id, listed.friend_id, NULL
FROM student s
JOIN friend_group g ON g.group_id = s.friend_group_id
CROSS JOIN LATERAL unnest(g.friends) AS listed(friend_id)
ON CONFLICT DO NOTHING;
```

Expected: a foreign key error, and **nothing** is inserted — not even the good rows. The whole statement is one piece of work: all of it happens or none of it does, the same rule you met as a transaction in lesson 11.

This is the old design's last word. It let bad data in without complaining; the new design will not accept it. Every real migration meets this, and the work of cleaning up is usually bigger than the change itself.

Run statement 3 to remove the ghost:

```sql
UPDATE friend_group
SET friends = ARRAY(
    SELECT id
    FROM unnest(friends) AS id
    WHERE id IN (SELECT student_id FROM student)
);
```

Then run the insert again. Expected: it succeeds. Check it against your prediction from step 1:

```sql
SELECT * FROM friendship ORDER BY student_id, friend_id;
```

## 3. Why `since` is empty

Every row you just created has no date. That is deliberate. The old design never recorded when a friendship started, so that fact does not exist anywhere. We could have written today's date into every row — and then the page would tell the students that every friendship in the school began on the day of this lesson.

An empty value in a database is written `NULL`, and it means *we do not know*. It is not the same as zero, and not the same as empty text.

**Checkpoint:** why is `NULL` a more honest answer here than today's date? What would go wrong a year from now if we had invented one?

Friendships made from now on will have a real date, because `POST /friend` will record the day it happens.

## 4. Point Go at the new table

Two changes in `server/main.go`.

**First**, the friends list now carries a date, so it needs its own response type. Add it next to `StudentResponse`:

```go
type FriendResponse struct {
	StudentID string  `json:"studentId"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Since     *string `json:"since"`
}
```

`*string` is a **pointer** to text. It can hold text, or it can hold nothing at all. A plain `string` cannot say "we do not know"; the best it can do is empty text, which is a different statement. When `Since` holds nothing, Go writes `null` in the JSON — the same word the database uses.

**Second**, replace the query inside `GET /friends`:

```go
rows, err := db.QueryContext(c.Request.Context(),
    `SELECT s.student_id, s.name, s.email, f.since
     FROM friendship f
     JOIN student s ON s.student_id = f.friend_id
     WHERE f.student_id = $1
     ORDER BY s.name`,
    studentID,
)
```

Compare it with the three-table version from lesson 11. It is shorter. There is no array, and `student` is only mentioned once: start from my friendships, and fetch each friend's record.

And the loop that reads the rows:

```go
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
```

| Code | Meaning |
| --- | --- |
| `sql.NullTime` | A date that might be `NULL`. Scanning a `NULL` into a plain date would fail. |
| `since.Valid` | True when the database gave a real date. |
| `since.Time.Format("2006-01-02")` | Write the date as `2026-02-01`. Go's date layout uses this one reference day, not letters like `YYYY-MM-DD`. |
| `&day` | Point `Since` at that text. If we skip this, `Since` stays empty and becomes `null`. |

## 5. Write to the new table

Replace the two `array_append` statements inside `POST /friend`:

```go
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
```

Add `"time"` to your imports. The transaction stays exactly as it was: two rows, both or neither.

`ON CONFLICT (student_id, friend_id) DO NOTHING` does the job that `NOT ($2 = ANY (friends))` did in lesson 11 — adding the same friendship twice changes nothing.

## 6. Delete code

`POST /student` still creates a `friend_group` row. There is no such table for much longer, so remove that `INSERT`, remove `groupID`, and put `friend_group_id` back out of the student insert:

```go
err = transaction.QueryRowContext(c.Request.Context(),
    `INSERT INTO student (student_id, name, email)
     VALUES ($1, $2, $3)
     RETURNING student_id, name, email`,
    request.StudentID, request.Name, request.Email,
).Scan(&student.StudentID, &student.Name, &student.Email)
```

Registration is now a single write again, so the transaction around it is no longer doing anything. You may leave it or take it out; discuss which you prefer and why.

Deleting code you wrote two lessons ago is part of the job, and a good sign. The feature it supported is still there — it just needs less code than it used to.

## 7. Check before you delete the old design

Run everything once more before dropping anything, because after the next step there is no going back:

| Request | Expected |
| --- | --- |
| `GET /friends?studentId=0231` | Arun, `"since": null` |
| `GET /friends?studentId=0235` | `[]` |
| `POST /friend` `{"studentId":"0231","friendId":"0235"}` | `201` |
| `GET /friends?studentId=0235` | Mali, with **today's** date |
| `GET /friends?studentId=9999` | `404` |

Only when all five are right, run the last two statements of the migration:

```sql
ALTER TABLE student DROP COLUMN friend_group_id;
DROP TABLE friend_group;
```

Restart Go and run the five checks again. Nothing should change — the old design was already unused.

## 8. Turn on the new page

Your teacher has a second version of the page, the one that shows dates. Copy it over the old one. From the project's top folder, not from **server**:

```sh
cp docs/week02/handouts/index-lesson-13.html web-app/index.html
```

The Go code does not change for this. `router.StaticFile("/", "../web-app/index.html")` has pointed at that same file since week 01 — you are replacing what the file says, not where the server looks for it.

Restart Go from **server** and look up `0231`. Each friend now shows a start date, and the friendships that came across from the arrays show "not recorded".

Look at the old page for a moment. It asked for `studentId`, `name` and `email` and ignored everything else, so it kept working perfectly while the API started sending a new `since` field. That is worth knowing: **adding** a field to an API is usually safe, because programs ignore what they were not looking for. **Removing** or **renaming** one is not — everything that was looking for it suddenly finds nothing.

**Checkpoint:** the office asks for one more thing: who introduced the two students. Where does that go now, and how much work is it compared with lesson 12? That difference is what the migration bought.

Next: [14. One big file is hard to read](14-one-big-file.md).
