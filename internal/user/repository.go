package user

import "context"

// Repository defines database access for users.
type Repository interface {
	List(ctx context.Context) ([]User, error)
	ChangePassword(ctx context.Context, userID int, password string) error
}
