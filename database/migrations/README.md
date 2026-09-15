# Migrations

`database/init/` runs **only when the database storage is empty**. It is the
starting point for a brand new computer.

`database/migrations/` holds the changes made *after* that starting point. Each
file is run once, in number order, on a database that already has records in it.

| File | Lesson | What it changes |
| --- | --- | --- |
| `002-add-friend-group.sql` | Week 02, lesson 10 | Adds `friend_group` and `student.friend_group_id`, and gives every existing student an empty friends list. |
| `003-friendship.sql` | Week 02, lesson 13 | Replaces the friends array with a `friendship` table that can record a date. |

Run them in DBeaver, one statement at a time, reading each one first.

A database is the one part of a program you cannot simply replace with a newer
version: it holds records people care about. That is why changes to it are kept
as small, ordered, readable steps.
