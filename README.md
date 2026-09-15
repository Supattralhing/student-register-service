# Student register: learning Go through a real task

Imagine you work at a school's front desk. Each week the office asks for one more thing, and you build it. By the end you are doing what a junior backend engineer does: changing a database that already holds records, keeping two tables in step, and proving your work still runs without clicking through it by hand.

| Week | The office asks for | Lessons |
| --- | --- | --- |
| [Week 01 — a register that remembers](docs/week01/README.md) | "Find a student, and register a new one." | 6–9 |
| [Week 02 — from a friends list to a friends table](docs/week02/README.md) | "Who are this student's friends? And when did they become friends?" | 10–15 |

Start at [week 01, lesson 6](docs/week01/06-start-the-database.md).

## What is in this project

```text
student-register-service/
├── compose.yaml               PostgreSQL in Docker
├── database/
│   ├── init/                  runs only on an empty database — the starting point
│   └── migrations/            the changes made after that — see its README
├── docs/
│   ├── week01/                lessons 6–9, plus the answer sheet
│   └── week02/                lessons 10–15, answer sheets, and the page handouts
├── server/                    the learner's Go code, one codebase all the way through
└── web-app/                   the page the API serves
```

`server/` is never copied or forked between weeks. It is one codebase that grows, exactly like a real project: week 02 edits code week 01 finished, and deletes some of it.

## Before teaching

Have Docker Desktop running, DBeaver Community installed, Postman available, and Go working. Downloading software can be a separate preparation session. `server/go.mod` requests Go `1.26.7`; run `go version` **inside `server`** before teaching and allow any toolchain download to finish.

The examples use `studentId`, `name`, and `email`. All names and addresses are fictional, and the database login is a public practice value for a database on your own computer.

Each week's README has its own "before teaching" notes. Read them.

## How to run each lesson

Use the same cycle every time: **predict → try → observe → explain**.

1. Describe one school-office task in everyday language.
2. Ask the learner what they expect to happen.
3. Let them type or change one small part and run it.
4. Compare what happened with their prediction.
5. Ask them to explain it without reading the code aloud.

Several lessons ask the learner to predict something and then be wrong on purpose. Those are the ones to slow down for, not skip.

Introduce words when they become useful. A **database** keeps records; a **table** is like one spreadsheet sheet; a **row** holds one student; a **column** holds one kind of information. **SQL** is the language used to ask the database to read or change records.

## Running everything

From the project's top folder:

```sh
docker compose up -d --wait
```

Then, in another terminal:

```sh
cd server
go run .
```

Open **http://127.0.0.1:8080/**. From week 02, lesson 15 onwards:

```sh
cd server
go test ./...
```
