package auth

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"` // never serialize password
}

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (us *UserStore) CreateUser(email, password string) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	query := `INSERT INTO auth_users (email, password_hash, created_at, updated_at) 
			  VALUES (?, ?, strftime('%s', 'now'), strftime('%s', 'now'))`
	
	result, err := us.db.Exec(query, email, string(hashedPassword))
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:    int(id),
		Email: email,
	}, nil
}

func (us *UserStore) ValidateUser(email, password string) (*User, error) {
	var user User
	var hashedPassword string

	query := `SELECT id, email, password_hash FROM auth_users WHERE email = ?`
	err := us.db.QueryRow(query, email).Scan(&user.ID, &user.Email, &hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	return &user, nil
}

func (us *UserStore) GetUser(id int) (*User, error) {
	var user User

	query := `SELECT id, email FROM auth_users WHERE id = ?`
	err := us.db.QueryRow(query, id).Scan(&user.ID, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}