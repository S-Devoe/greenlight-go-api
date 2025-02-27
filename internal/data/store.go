package data

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Movies      MovieStore
	Users       UserStore
	Tokens      TokenStore
	Permissions PermissionStore
}

func NewStore(db *pgxpool.Pool) Store {
	return Store{
		Movies:      MovieStore{DB: db},
		Users:       UserStore{DB: db},
		Tokens:      TokenStore{DB: db},
		Permissions: PermissionStore{DB: db},
	}
}

// func NewMockStore() Store {
// 	return Store{
// 		Movies: MockMovieStore{},
// 		Users:  MockUserStore{},
// 	}
// }
