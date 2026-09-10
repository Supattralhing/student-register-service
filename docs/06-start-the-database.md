# 6. Give the student register a place to keep records

**Today's result:** PostgreSQL is running on your computer and contains two sample students.

So far, your API has used an answer written directly in the Go code. To keep new registrations, we need somewhere to save them. Think of the database as the school's student register.

## Meet the two new tools

**PostgreSQL**, also called **Postgres**, is the program that stores and finds records. A database *server* is that program running and waiting for requests; it can run on your own laptop.

**Docker** starts PostgreSQL in its own small workspace, called a **container**. **Docker Compose** reads a setup file so everyone in the lesson can start the database the same way. The teacher supplies this setup; you do not need to memorize it.

Install and open [Docker Desktop](https://docs.docker.com/compose/install/), which includes Docker Compose. Wait until Docker says it is running.

## 1. Find the setup files

Open the project folder in your editor. These files are supplied:

```text
student-register-service/
├── compose.yaml
├── database/
│   └── init/
│       └── 01-students.sql
└── server/
    ├── go.mod
    └── main.go
```

Open [compose.yaml](../compose.yaml). Its main instructions mean:

| Setting | Everyday meaning |
| --- | --- |
| `image: postgres:17-alpine` | Use this packaged version of PostgreSQL. |
| `POSTGRES_DB` | Name our database `student_register`. |
| `POSTGRES_USER` and `POSTGRES_PASSWORD` | Create the practice login we will use. |
| `127.0.0.1:5434:5432` | Let programs on this computer reach the database through door number `5434`; Docker forwards them to PostgreSQL's door `5432`. |
| `student_data` | Keep the saved records in storage that survives replacing the container. Docker calls this a **volume**. |
| `./database/init` | Run the supplied setup instructions when creating the database for the first time. |
| `healthcheck` | Ask regularly whether PostgreSQL is ready to accept a connection. |

`127.0.0.1` means “this computer.” A **port** is like a numbered door to a program. Our database uses port **5434** on your computer; our API will use **8080**. They do different jobs.

This setup uses PostgreSQL 17 and its data folder `/var/lib/postgresql/data`. Keep the version and folder together: PostgreSQL 18 images use a different folder layout. The startup script runs only with empty database storage. [Docker's PostgreSQL image documentation](https://hub.docker.com/_/postgres) explains both behaviors.

## 2. Read the student table setup

Open [database/init/01-students.sql](../database/init/01-students.sql):

```sql
CREATE TABLE student (
    student_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL
);

INSERT INTO student (student_id, name, email)
VALUES
    ('0231', 'Mali Srisuk', 'mali@example.com'),
    ('0232', 'Arun Jaidee', 'arun@example.com');
```

Read it like this: “Create a sheet called `student`. Each student has an ID, a name, and an email address. Then add these two students.”

| SQL words | Meaning |
| --- | --- |
| `CREATE TABLE` | Create the structure that holds our records. |
| `TEXT` | Store characters, such as a name or an ID. |
| `PRIMARY KEY` | This field identifies a row. Two students cannot use the same ID, and it cannot be missing. |
| `NOT NULL` | A value must be supplied. This alone does not reject empty text or check an email address. |
| `INSERT INTO ... VALUES` | Add rows containing these values. |

An ID is a label, like a product code. We do not add student IDs together. Keeping `0231` as text preserves the first zero. SQL puts text inside single quotes: `'0231'`.

**Predict:** How many rows will the new table contain? Which part prevents two students from sharing an ID?

## 3. Start PostgreSQL

Open a terminal in the **project's top folder**, where `compose.yaml` is. A terminal is a place to type instructions for your computer. If you are currently inside `server`, type `cd ..` to move up one folder.

Run these commands one at a time:

```sh
docker compose version
docker compose up -d --wait
docker compose ps
```

The first command should print a version number. The second starts the database; the first run may download PostgreSQL. `-d` keeps it running in the background, and `--wait` waits for it to become ready. The last command shows its state. Look for `db` and **healthy**. [Docker's command reference](https://docs.docker.com/reference/cli/docker/compose/up/) describes these options.

If it does not become healthy, read its messages:

```sh
docker compose logs --tail=50 db
```

## 4. Check that the setup worked

Still in the **project's top folder**, run:

```sh
docker compose exec db psql -U student_user -d student_register -c "SELECT student_id, name, email FROM student ORDER BY student_id;"
```

This asks PostgreSQL's small command-line tool, `psql`, to show the students. You can copy this check; next lesson uses a visual tool.

Expected records:

| student_id | name | email |
| --- | --- | --- |
| 0231 | Mali Srisuk | mali@example.com |
| 0232 | Arun Jaidee | arun@example.com |

## 5. Save this connection card

Use the same details in DBeaver and Go:

| Field | Value |
| --- | --- |
| Host | `127.0.0.1` |
| Port | `5434` |
| Database | `student_register` |
| Username | `student_user` |
| Password | `student_password` |

The values are for this local exercise. Go and DBeaver run outside Docker, so both use port `5434`.

## Stop for today, or continue

To pause the database without removing its records, run this from the **project's top folder**:

```sh
docker compose stop
```

To use it again:

```sh
docker compose up -d --wait
```

Editing `01-students.sql` later does **not** change an existing database when you restart it. That file is the first-time setup. We will add new records with SQL or the API.

Teacher-only fresh start, **only when you intend to erase all practice students saved by this Compose project**:

```sh
docker compose down -v
docker compose up -d --wait
```

`-v` deletes the saved volume. This is an optional reset, not the normal way to stop for the day.

## If something looks different

| What you see | What to check |
| --- | --- |
| `docker: command not found` | Install Docker Desktop, then reopen your terminal. |
| Cannot connect to the Docker daemon | Open Docker Desktop and wait until it is running. |
| No configuration file found | Move to the folder containing `compose.yaml`. |
| Port `5434` is already allocated | Another program is using that door. With your teacher, choose a free port in `compose.yaml`, then use it in both DBeaver and the Go connection text in lesson 8. |
| Image download fails | Check the internet connection and Docker's error message, then retry. |
| No `student` table after startup | Read the database logs. The SQL may have failed, or this volume may already have been initialized. Do not erase saved work just to try again. |

**Ready to continue:** You can show both sample students and explain what PostgreSQL, Docker, and a volume each do.

Next: [7. Open the register in DBeaver](07-explore-with-dbeaver.md).
