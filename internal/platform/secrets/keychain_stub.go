//go:build !darwin

package secrets

import (
	"context"
	"errors"
)

type KeychainStore struct{}

func NewKeychainStore(string) *KeychainStore { return &KeychainStore{} }

func (s *KeychainStore) Get(ctx context.Context, name string) (string, error) {
	_ = ctx
	_ = name
	return "", errors.New("keychain store unavailable")
}

func (s *KeychainStore) Set(ctx context.Context, name, value string) error {
	_ = ctx
	_ = name
	_ = value
	return errors.New("keychain store unavailable")
}

func (s *KeychainStore) Delete(ctx context.Context, name string) error {
	_ = ctx
	_ = name
	return errors.New("keychain store unavailable")
}
