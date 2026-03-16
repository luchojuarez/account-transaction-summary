// Package repository provides TransactionRepository and UserRepository adapters.
// This file implements UserRepository with a simple hardcoded map.
package repository

import "errors"

// Hardcoded user ID -> email. Name defaults to "Valued Customer" when not set.
var defaultUsers = map[int]struct {
	email string
	name  string
}{
	1: {email: "lucho.juarez79@gmail.com", name: "Luciano Juarez"},
}

// InMemoryUserRepository implements ports.UserRepository with a hardcoded map.
type InMemoryUserRepository struct{}

// NewInMemoryUserRepository creates a UserRepository with hardcoded users.
func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{}
}

// GetUser implements ports.UserRepository.
func (r *InMemoryUserRepository) GetUser(userID int) (email, name string, err error) {
	u, ok := defaultUsers[userID]
	if !ok {
		return "", "", errors.New("user not found")
	}
	name = u.name
	if name == "" {
		name = "Valued Customer"
	}
	return u.email, name, nil
}
