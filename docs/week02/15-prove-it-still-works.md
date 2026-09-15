# 15. Prove it still works tomorrow

**Today's result:** `go test ./...` checks the whole friends feature by itself, in about a second.

Every check you have run this week you ran by hand: type an ID into Postman, look at the answer, decide whether it is right. That works until there are twenty checks and you have changed one line. Then you run three of them, decide the rest are probably fine, and ship a bug.

A **test** is those same checks written as code, so the computer runs all of them, every time, and never decides something is probably fine.

## 1. How Go finds tests

| Rule | Meaning |
| --- | --- |
| The file name ends in `_test.go` | Go only builds it when testing, never in `go run .`. |
| The function starts with `Test` and takes `*testing.T` | Go runs it as a test. |
| `t.Errorf(...)` | Report a failure and keep going. |
| `t.Fatalf(...)` | Report a failure and stop this test now. |
| `go test ./...` | Run every test in the project. |

Create `server/friends_test.go`. Everything today goes in that one file.

## 2. The small kind: a unit test

Lesson 14 pulled `validateFriendPair` out on its own. It needs no database and no HTTP — two pieces of text in, an answer out. That makes it the easiest thing in the project to test.

```go
func TestValidateFriendPair(t *testing.T) {
	cases := []struct {
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

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateFriendPair(testCase.studentID, testCase.friendID)
			if testCase.wantError && err == nil {
				t.Errorf("expected a problem to be reported, got none")
			}
			if !testCase.wantError && err != nil {
				t.Errorf("expected no problem, got: %v", err)
			}
		})
	}
}
```

This shape is called a **table-driven test**: the cases are a list, and one small piece of code runs all of them. Adding a fifth case is one line. `t.Run` gives each case its own name, so a failure tells you *which* case broke.

Run it:

```sh
cd server
go test ./...
```

**Predict first:** open `handler.go` and delete the `studentID == friendID` check. What will `go test ./...` say? Try it, read the output, then put the check back.

## 3. The bigger kind: an integration test

A unit test checks one piece on its own. It cannot catch a mistake in your SQL, because it never touches the database.

So the other test does what Postman does: it uses the real database in Docker.

```go
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
```

| Code | Meaning |
| --- | --- |
| `databaseURL()` | The same function `main` uses, from lesson 14. The test can be pointed at another database by setting `DATABASE_URL`. |
| `t.Skip` | Say "not run" rather than "failed" when the database is not started. A missing database is not a broken API. |
| `t.Cleanup` | Run this when the test finishes, whether it passed or failed. |
| `LIKE 'test_%'` | Only rows this test created. |
| `t.Helper()` | Tell Go this is a helper, so failures point at the real test line. |

**A test that leaves a mess behind is worse than no test.** It would fill the school register with invented students, and the second run would fail because they already exist. That is why every test student starts with `test_` and why the cleanup deletes `friendship` rows before `student` rows — the foreign keys from lesson 12 will not allow a student to be deleted while a friendship still points at them. Your own schema is now stopping you doing something careless.

Next, a way to call the API without Postman:

```go
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
```

`httptest` makes a request and catches the reply in memory. No port is opened, nothing is running in another terminal. This is Postman, in code.

## 4. The test worth writing

```go
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

	// The point of the test: a friendship must be visible from both sides.
	for _, pair := range [][2]string{{mali, arun}, {arun, mali}} {
		response := call(router, http.MethodGet, "/friends?studentId="+pair[0], "")
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
```

Read what this test is really saying: *one request was sent, and both students must be able to see the result.* That is the rule the transaction in lesson 11 exists to protect, and it is the rule most likely to break quietly if someone edits that code later.

**Predict first:** in `store.go`, comment out the **second** `ExecContext` in `AddFriend`, so only one direction is written. Which check fails, and what does the message say? This is the bug the test exists to catch. Put the line back afterwards.

The other two tests are short, and cover the distinction from lesson 11:

```go
func TestFriendsOfAStudentWithNoFriends(t *testing.T) {
	db := openTestDatabase(t)
	store := NewStore(db)
	router := newTestRouter(store)

	lonely := testPrefix + "lonely"
	if _, err := store.CreateStudent(t.Context(), lonely, "Test Lonely", "lonely@example.com"); err != nil {
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
	router := newTestRouter(NewStore(db))

	response := call(router, http.MethodGet, "/friends?studentId="+testPrefix+"nobody", "")
	if response.Code != http.StatusNotFound {
		t.Errorf("expected 404 for a student who does not exist, got %d", response.Code)
	}
}
```

The whole file, with its imports, is in [reference/lesson-15/friends_test.go.txt](reference/lesson-15/friends_test.go.txt).

## 5. Run everything

```sh
docker compose up -d --wait   # from the project's top folder
cd server
go test ./...
```

Expected:

```text
ok  	student-register-service	0.9s
```

Add `-v` to see each test name as it runs:

```sh
go test -v ./...
```

**Checkpoint:** in DBeaver, run `SELECT * FROM student WHERE student_id LIKE 'test_%';` after the tests have finished. Expected: no rows. Explain why that matters.

## 6. What you can now say about your own work

Before this lesson, "it works" meant you tried it. Now it means the computer checked it, and will check it again every time anyone changes anything.

The two kinds are worth naming clearly:

| | Unit test | Integration test |
| --- | --- | --- |
| Checks | One function on its own | Several parts working together |
| Needs the database | No | Yes |
| Speed | Instant | Fast, but not instant |
| Catches | Wrong rules and wrong logic | Wrong SQL, wrong status codes, wrong wiring |

You need both. A project with only unit tests passes every test and cannot talk to its database.

## 7. The demo

Do this in front of your teacher, start to finish, without notes:

1. Start the database and the API.
2. Open the page and look up `0231`. Mali's friends appear with dates.
3. Add Somchai (`0234`) as a friend of Arun (`0232`).
4. Look up `0231` again. Mali's list has **not** changed.
5. Run `go test ./...`. Everything passes.
6. Explain, in your own words, why `since` could not exist before lesson 12, and why Mali cannot see Somchai.

That is the week.

## Where this goes next

Things you now have good reasons to want, and will build in week 03:

| Want | Because |
| --- | --- |
| A migration tool | Lesson 10 left two ways to reach the same database. |
| Packages, not just files | Lesson 14's three files will become three folders. |
| Fake stores and interfaces | Lesson 15's tests need Docker running. Some should not. |
| `DELETE /friend` | Friendships end. Ours cannot. |
| Indexes | `WHERE student_id = $1` is fast with five students. Ask again at fifty thousand. |

Back to the [week 02 lessons](README.md), or the [course index](../../README.md).
