package store

import (
	"errors"
	"sync"
)

type Flag struct {
	ID             int    `json:"id"`
	Key            string `json:"key"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description"`
	RolloutPercent int    `json:"rollout_percent"`
}

var (
	ErrConflict = errors.New("flag already exists")
	ErrNotFound = errors.New("flag not found")
)

type Store struct {
	mu     sync.RWMutex
	flags  map[string]Flag
	nextID int
}

func New() *Store {
	return &Store{
		flags:  make(map[string]Flag),
		nextID: 1,
	}
}

func (s *Store) Create(key string, enabled bool, description string, rolloutPercent int) (Flag, error) {
	return Flag{}, nil
}

func (s *Store) Get(key string) (Flag, error) {
	return Flag{}, nil
}

func (s *Store) List() []Flag {
	return nil
}

func (s *Store) Update(key string, enabled *bool, description *string, rolloutPercent *int) (Flag, error) {
	return Flag{}, nil
}

func (s *Store) Delete(key string) error {
	return nil
}
