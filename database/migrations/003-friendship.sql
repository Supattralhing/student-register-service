-- Week 02, lesson 13: move friendships out of the array and into their own table.
-- Run these statements in DBeaver, one at a time.

-- 1. The new table. Both columns point at a real student, so a friendship can
--    never mention someone who does not exist. A friendship also has a date now.
CREATE TABLE friendship (
    student_id TEXT NOT NULL REFERENCES student (student_id),
    friend_id  TEXT NOT NULL REFERENCES student (student_id),
    since      DATE,
    PRIMARY KEY (student_id, friend_id),
    CHECK (student_id <> friend_id)
);

-- 2. Look for ghosts first: IDs sitting in an array that are not real students.
--    `unnest` turns one array into one row per element. If this returns any
--    rows, statement 4 will refuse to run.
SELECT s.student_id AS listed_by, listed.friend_id AS ghost_id
FROM student s
JOIN friend_group g ON g.group_id = s.friend_group_id
CROSS JOIN LATERAL unnest(g.friends) AS listed(friend_id)
WHERE listed.friend_id NOT IN (SELECT student_id FROM student);

-- 3. Remove any ghost found above. Run this only if statement 2 returned rows.
UPDATE friend_group
SET friends = ARRAY(
    SELECT id
    FROM unnest(friends) AS id
    WHERE id IN (SELECT student_id FROM student)
);

-- 4. One array element becomes one row. `since` is left empty on purpose: the
--    old design never recorded a date, so we must not invent one.
INSERT INTO friendship (student_id, friend_id, since)
SELECT s.student_id, listed.friend_id, NULL
FROM student s
JOIN friend_group g ON g.group_id = s.friend_group_id
CROSS JOIN LATERAL unnest(g.friends) AS listed(friend_id)
ON CONFLICT DO NOTHING;

-- 5. Check the move before deleting anything.
SELECT * FROM friendship ORDER BY student_id, friend_id;

-- 6. The old design is no longer used. Remove it, column first, then table.
ALTER TABLE student DROP COLUMN friend_group_id;
DROP TABLE friend_group;
