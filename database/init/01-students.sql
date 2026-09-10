-- One row represents one student. IDs are text so 0231 keeps its first zero.
CREATE TABLE student (
    student_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL
);

INSERT INTO student (student_id, name, email)
VALUES
    ('0231', 'Mali Srisuk', 'mali@example.com'),
    ('0232', 'Arun Jaidee', 'arun@example.com');
