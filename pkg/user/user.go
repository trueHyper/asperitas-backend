package user

type User struct {
	Username string `json:"username"`
	ID       string `json:"id"`
	Password string `json:"-" bson:"-"`
}

type Repository interface {
	Create(user *User) error
	FindByUsername(username string) (*User, error)
}
