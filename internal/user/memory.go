package user

import "context"

// MemoryRepository is in-memory implementation useful for tests.
type MemoryRepository struct {
	users map[int]User
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{users: map[int]User{1: {ID: 1, Username: "alice", Password: "secret"}}}
}

func (r *MemoryRepository) List(ctx context.Context) ([]User, error) {
	var res []User
	for _, u := range r.users {
		res = append(res, u)
	}
	return res, nil
}

func (r *MemoryRepository) ChangePassword(ctx context.Context, userID int, password string) error {
	u, ok := r.users[userID]
	if !ok {
		return nil
	}
	u.Password = password
	r.users[userID] = u
	return nil
}
