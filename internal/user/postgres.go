package user

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) List(ctx context.Context) ([]User, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, username, password FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Password); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *PostgresRepository) ChangePassword(ctx context.Context, userID int, password string) error {
	_, err := r.pool.Exec(ctx, "UPDATE users SET password=$1 WHERE id=$2", password, userID)
	return err
}
