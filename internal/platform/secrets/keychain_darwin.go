//go:build darwin

package secrets

import (
	"context"
	"errors"

	keychain "github.com/keybase/go-keychain"
)

const defaultServiceName = "com.wails.agent-skills-manager.ai"

type KeychainStore struct {
	service string
	label   string
}

func NewKeychainStore(service string) *KeychainStore {
	if service == "" {
		service = defaultServiceName
	}
	return &KeychainStore{
		service: service,
		label:   "Agent Skills Manager AI Key",
	}
}

func (s *KeychainStore) Get(ctx context.Context, name string) (string, error) {
	_ = ctx
	password, err := keychain.GetGenericPassword(s.service, name, s.label, "")
	if err != nil {
		if errors.Is(err, keychain.ErrorItemNotFound) {
			return "", nil
		}
		return "", err
	}
	return string(password), nil
}

func (s *KeychainStore) Set(ctx context.Context, name, value string) error {
	_ = ctx
	item := keychain.NewGenericPassword(s.service, name, s.label, []byte(value), "")
	item.SetAccessible(keychain.AccessibleWhenUnlockedThisDeviceOnly)
	item.SetSynchronizable(keychain.SynchronizableNo)

	if err := keychain.AddItem(item); err == nil {
		return nil
	} else if !errors.Is(err, keychain.ErrorDuplicateItem) {
		return err
	}

	if err := keychain.DeleteGenericPasswordItem(s.service, name); err != nil && !errors.Is(err, keychain.ErrorItemNotFound) {
		return err
	}
	return keychain.AddItem(item)
}

func (s *KeychainStore) Delete(ctx context.Context, name string) error {
	_ = ctx
	err := keychain.DeleteGenericPasswordItem(s.service, name)
	if errors.Is(err, keychain.ErrorItemNotFound) {
		return nil
	}
	return err
}
