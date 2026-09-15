-- Week 02, lesson 10: give every student a place to keep a friends list.
-- Run these statements in DBeaver, one at a time, on your existing database.

-- 1. One row per student's friends list.
CREATE TABLE friend_group (
    group_id TEXT PRIMARY KEY,
    friends  TEXT[] NOT NULL DEFAULT '{}'
);

-- 2. Each student points at their own list. The column allows NULL for a moment,
--    because the students already in the table do not have a list yet.
ALTER TABLE student
    ADD COLUMN friend_group_id TEXT REFERENCES friend_group (group_id);

-- 3. Two more students to be friends with. Skipped if they are already there.
INSERT INTO student (student_id, name, email)
VALUES
    ('0234', 'Somchai Wong', 'somchai@example.com'),
    ('0235', 'Nida Chai', 'nida@example.com')
ON CONFLICT (student_id) DO NOTHING;

-- 4. Backfill. Every student who exists right now gets an empty list named after
--    their own ID, including any student you registered yourself in lesson 9.
INSERT INTO friend_group (group_id)
SELECT 'g-' || student_id
FROM student
WHERE friend_group_id IS NULL;

UPDATE student
SET friend_group_id = 'g-' || student_id
WHERE friend_group_id IS NULL;

-- 5. Check: this must return 0.
SELECT count(*) AS students_without_a_list FROM student WHERE friend_group_id IS NULL;
