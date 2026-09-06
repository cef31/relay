package domain

import "time"

type Person struct {
	Id          string
	PubKey      string
	Name        string
	PhoneNumber string
	Email       string
	BirthDate   time.Time
	CreatedAt   time.Time
}
