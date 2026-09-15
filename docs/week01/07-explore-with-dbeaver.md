# 7. Open the student register in DBeaver

**Today's result:** You can see the saved students and find one by ID without using Go.

PostgreSQL keeps the records. **DBeaver** is an app that lets you look at them, much like opening a spreadsheet. Installing DBeaver does not install or start PostgreSQL. Start the database from lesson 6 before connecting.

## 1. Connect to the practice database

Install [DBeaver Community](https://dbeaver.io/download/) and open it.

1. Choose **Database → New Database Connection**.
2. Search for **PostgreSQL**, select it, and click **Next**.
3. Enter the values below. If offered a choice, connect using **Host** with username/password authentication.
4. Click **Test Connection**. If prompted to download driver files, allow the download. A **driver** is the helper DBeaver uses to talk to PostgreSQL.
5. Wait for a successful connection message, then click **Finish**.

| DBeaver field | Enter |
| --- | --- |
| Host | `127.0.0.1` |
| Port | `5434` |
| Database | `student_register` |
| Username | `student_user` |
| Password | `student_password` |

The button names can vary slightly by version. These steps follow DBeaver's [connection guide](https://dbeaver.com/docs/dbeaver/Create-Connection/) and [PostgreSQL settings](https://dbeaver.com/docs/dbeaver/Database-driver-PostgreSQL/).

If the test fails, check that `docker compose ps` shows a healthy database, and compare every field with the table above. Use port `5434`, including when DBeaver suggests `5432`.

## 2. Look at the students

In the left sidebar, expand your connection. It may be called **Database Navigator** or **Connections**. Find:

```text
student_register
└── Schemas
    └── public
        └── Tables
            └── student
```

Some versions show an extra **Databases → student_register** level. A **schema** is a folder-like grouping for tables. Use the supplied `public` grouping today.

Double-click `student`, then select the **Data** tab. You should see Mali and Arun. Use the refresh button if the view is stale. DBeaver's [Data Editor](https://dbeaver.com/docs/dbeaver/Data-Editor/) displays table records in a grid.

Point to one row and say what it represents. Then point to each column and explain what it stores. Notice that Mali's ID is `0231`, including its first zero.

## 3. Ask a question with SQL

Right-click your connection and choose **SQL Editor → New SQL Script**. Make sure the editor is connected to `student_register`.

Type:

```sql
SELECT student_id, name, email
FROM student
ORDER BY student_id;
```

Put the cursor inside this statement and use **SQL Editor → Execute SQL Statement**, or the matching toolbar button. The result appears below the editor. DBeaver can run either the current statement or selected text; select only the example you want to run. See its [SQL execution guide](https://dbeaver.com/docs/dbeaver/SQL-Execution/).

Read the SQL as: “Show the ID, name, and email from the student register, sorted by ID.” SQL instructions usually end with `;`.

Now replace it with:

```sql
SELECT student_id, name, email
FROM student
WHERE student_id = '0231';
```

`WHERE` means “keep only rows that match this condition.” You should see only Mali. `SELECT` reads records; running it does not change the table.

**Try it yourself:** Change only `'0231'` to `'0232'`. Predict the name before running it. Then try `'9999'`. An empty result means no student matched; it does not mean the database is broken.

## 4. Change one practice value and see what refresh does

Return to the table's **Data** tab. Double-click Mali's name cell and change it to `Mali Srisuk (practice)`.

Click the Data Editor's **Save** button. If DBeaver shows the SQL it plans to run, check that it changes only Mali's name, then apply it. If your connection uses **Manual commit**, also click **Commit** to finish saving. Commit means “make this change final so other connections can see it.” In **Auto-commit** mode, no extra commit is needed. These modes are explained in DBeaver's [transaction documentation](https://dbeaver.com/docs/dbeaver/Auto-and-Manual-Commit-Modes/).

Run the ID lookup again in the SQL editor. It should show the edited name. Change the name back to `Mali Srisuk` in the grid and save it the same way, so later examples match.

DBeaver may keep an older result on screen until you rerun the SQL or refresh the Data tab. Looking at an old result does not ask the database for a new one.

## 5. Explain the difference between the apps

| App | What it does in this course |
| --- | --- |
| Postman | Sends a request to our Go API and shows the reply. |
| Go API | Reads that request, talks to PostgreSQL, and prepares a reply. |
| DBeaver | Talks directly to PostgreSQL so we can inspect the records. |
| PostgreSQL | Stores the records and carries out SQL instructions. |

**Predict and check:** Close DBeaver and reopen it. Reconnect and look at the students. Did closing the viewing app remove the records?

**Ready to continue:** Without help, connect to the database, find `0232`, and explain why searching for `9999` produces no rows. You can also save a practice edit and restore it.

Next: [8. Make Go find a student](08-find-a-student.md).
