package main

import (
	"context"
	"database/sql"
	"time"
)

// Store is the only part of the program that writes SQL. Everything else asks
// Store for what it needs.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) FindStudent(ctx context.Context, studentID string) (StudentResponse, error) {
	var student StudentResponse
	err := s.db.QueryRowContext(ctx,
		"SELECT student_id, name, email FROM student WHERE student_id = $1",
		studentID,
	).Scan(&student.StudentID, &student.Name, &student.Email)
	return student, err
}

func (s *Store) CreateStudent(ctx context.Context, studentID string, name string, email string) (StudentResponse, error) {
	var student StudentResponse
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO student (student_id, name, email)
     		VALUES ($1, $2, $3)
     		RETURNING student_id, name, email`,
		studentID, name, email,
	).Scan(&student.StudentID, &student.Name, &student.Email)

	return student, err
}

func (s *Store) studentExists(ctx context.Context, studentID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM student WHERE student_id = $1)",
		studentID,
	).Scan(&exists)
	return exists, err
}

func (s *Store) ListFriends(ctx context.Context, studentID string) ([]FriendResponse, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT s.student_id, s.name, s.email, f.since
     		FROM friendship f
     		JOIN student s ON s.student_id = f.friend_id
    		WHERE f.student_id = $1
     		ORDER BY s.name`,
		studentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	friends := []FriendResponse{}
	for rows.Next() {
		var friend FriendResponse
		var since sql.NullTime
		if err := rows.Scan(&friend.StudentID, &friend.Name, &friend.Email, &since); err != nil {
			return nil, err
		}
		if since.Valid {
			day := since.Time.Format("2006-01-02")
			friend.Since = &day
		}
		friends = append(friends, friend)
	}
	return friends, rows.Err()
}

func (s *Store) AddFriend(ctx context.Context, studentID string, friendID string) error {
	transaction, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()

	today := time.Now().Format("2006-01-02")
	const addOneDirection = `INSERT INTO friendship (student_id, friend_id, since)
                         VALUES ($1, $2, $3)
                         ON CONFLICT (student_id, friend_id) DO NOTHING`
	if _, err := transaction.ExecContext(ctx, addOneDirection, studentID, friendID, today); err != nil {
		return err
	}

	if _, err := transaction.ExecContext(ctx, addOneDirection, friendID, studentID, today); err != nil {
		return err
	}
	return transaction.Commit()
}
