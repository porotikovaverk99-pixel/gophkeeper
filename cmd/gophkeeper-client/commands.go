package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/crypto"
	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/api"
	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/storage"
	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/vault"
)

func registerCmd() *cobra.Command {
	var login string

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if login == "" {
				return fmt.Errorf("укажите логин: --login")
			}

			password, err := readPassword("Account password: ")
			if err != nil {
				return err
			}
			if password == "" {
				return fmt.Errorf("пароль аккаунта не может быть пустым")
			}

			masterPassword, err := readPassword("Master password: ")
			if err != nil {
				return err
			}
			if masterPassword == "" {
				return fmt.Errorf("мастер-пароль не может быть пустым")
			}

			client := api.NewClient(serverAddr)
			token, err := client.Register(context.Background(), login, password)
			if err != nil {
				return err
			}

			localStorage, err := storage.OpenLocalStorage(storagePath)
			if err != nil {
				return err
			}

			vaultManager, salt, err := vault.NewManagerWithSalt(masterPassword)
			if err != nil {
				return err
			}

			verifier, err := vaultManager.CreateVerifier()
			if err != nil {
				return err
			}

			localStorage.Token = token
			localStorage.Login = login
			localStorage.MasterSalt = salt
			localStorage.MasterVerifier = verifier
			localStorage.LastSync = time.Time{}

			if err := localStorage.Save(); err != nil {
				return err
			}

			printSuccess("Пользователь %q успешно зарегистрирован", login)
			return nil
		},
	}

	cmd.Flags().StringVar(&login, "login", "", "user login")
	return cmd
}

func loginCmd() *cobra.Command {
	var login string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to GophKeeper server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if login == "" {
				return fmt.Errorf("укажите логин: --login")
			}

			password, err := readPassword("Account password: ")
			if err != nil {
				return err
			}
			if password == "" {
				return fmt.Errorf("пароль аккаунта не может быть пустым")
			}

			masterPassword, err := readPassword("Master password: ")
			if err != nil {
				return err
			}
			if masterPassword == "" {
				return fmt.Errorf("мастер-пароль не может быть пустым")
			}

			client := api.NewClient(serverAddr)
			token, err := client.Login(context.Background(), login, password)
			if err != nil {
				return err
			}

			localStorage, err := storage.OpenLocalStorage(storagePath)
			if err != nil {
				return err
			}

			if localStorage.MasterSalt == "" {
				vaultManager, salt, err := vault.NewManagerWithSalt(masterPassword)
				if err != nil {
					return err
				}
				verifier, err := vaultManager.CreateVerifier()
				if err != nil {
					return err
				}
				localStorage.MasterSalt = salt
				localStorage.MasterVerifier = verifier
			} else {
				vaultManager, err := vault.NewManager(masterPassword, localStorage.MasterSalt)
				if err != nil {
					return err
				}
				if err := validateMasterPassword(vaultManager, localStorage); err != nil {
					return err
				}
				if localStorage.MasterVerifier == "" {
					verifier, err := vaultManager.CreateVerifier()
					if err != nil {
						return err
					}
					localStorage.MasterVerifier = verifier
				}
			}

			localStorage.Token = token
			localStorage.Login = login

			if err := localStorage.Save(); err != nil {
				return err
			}

			printSuccess("Вход выполнен: %q", login)
			return nil
		},
	}

	cmd.Flags().StringVar(&login, "login", "", "user login")
	return cmd
}

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Synchronize local data with server",
		RunE: func(cmd *cobra.Command, args []string) error {
			localStorage, client, err := openSession()
			if err != nil {
				return err
			}

			entries, err := client.SyncEntries(context.Background(), localStorage.LastSync)
			if err != nil {
				return err
			}

			for _, entry := range entries {
				localStorage.UpsertEntry(entry)
			}

			localStorage.LastSync = time.Now().UTC()
			if err := localStorage.Save(); err != nil {
				return err
			}

			printSuccess("Синхронизация завершена: получено записей — %d", len(entries))
			return nil
		},
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			localStorage, _, err := openSession()
			if err != nil {
				return err
			}

			entries := localStorage.ListEntries()
			if len(entries) == 0 {
				printSuccess("Записей пока нет")
				return nil
			}

			for _, entry := range entries {
				fmt.Printf("%s  type=%s  metadata=%q  version=%d\n", entry.ID, entry.Type, entry.Metadata, entry.Version)
			}

			return nil
		},
	}
}

func openSession() (*storage.LocalStorage, *api.Client, error) {
	localStorage, err := storage.OpenLocalStorage(storagePath)
	if err != nil {
		return nil, nil, err
	}

	if localStorage.Token == "" {
		return nil, nil, fmt.Errorf("not authenticated, run login or register first")
	}

	client := api.NewClient(serverAddr)
	client.SetToken(localStorage.Token)

	return localStorage, client, nil
}

func openVault() (*storage.LocalStorage, *api.Client, *vault.Manager, error) {
	localStorage, client, err := openSession()
	if err != nil {
		return nil, nil, nil, err
	}

	masterPassword, err := readPassword("Master password: ")
	if err != nil {
		return nil, nil, nil, err
	}
	if masterPassword == "" {
		return nil, nil, nil, fmt.Errorf("мастер-пароль не может быть пустым")
	}

	vaultManager, err := vault.NewManager(masterPassword, localStorage.MasterSalt)
	if err != nil {
		return nil, nil, nil, err
	}

	if err := validateMasterPassword(vaultManager, localStorage); err != nil {
		return nil, nil, nil, err
	}

	if localStorage.MasterVerifier == "" {
		verifier, err := vaultManager.CreateVerifier()
		if err != nil {
			return nil, nil, nil, err
		}
		localStorage.MasterVerifier = verifier
		if err := localStorage.Save(); err != nil {
			return nil, nil, nil, err
		}
	}

	return localStorage, client, vaultManager, nil
}

func validateMasterPassword(vaultManager *vault.Manager, localStorage *storage.LocalStorage) error {
	if localStorage.MasterVerifier != "" {
		return vaultManager.ValidateVerifier(localStorage.MasterVerifier)
	}

	for _, entry := range localStorage.Entries {
		if entry.Deleted {
			continue
		}
		var probe struct{}
		if err := vaultManager.DecryptPayload(entry.Payload, &probe); err != nil {
			if errors.Is(err, crypto.ErrInvalidCiphertext) {
				return vault.ErrWrongMasterPassword
			}
			return err
		}
		return nil
	}

	return nil
}

func decryptPayload(vaultManager *vault.Manager, payload []byte, target any) error {
	if err := vaultManager.DecryptPayload(payload, target); err != nil {
		if errors.Is(err, crypto.ErrInvalidCiphertext) {
			return vault.ErrWrongMasterPassword
		}
		return err
	}
	return nil
}

func readPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stdout, prompt)

	if term.IsTerminal(int(syscall.Stdin)) {
		bytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}

	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(value), nil
}
