package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/api"
	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/storage"
	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/vault"
)

func updateCmd() *cobra.Command {
	var metadata, loginName, password, text, number, holder, expiry, cvv, filePath string

	cmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update entry by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entryID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("некорректный id записи")
			}

			if !hasUpdateFlags(cmd) {
				return fmt.Errorf("укажите хотя бы одно поле для обновления")
			}

			localStorage, client, vaultManager, err := openVault()
			if err != nil {
				return err
			}

			entry, err := loadEntry(localStorage, client, entryID)
			if err != nil {
				return err
			}

			if cmd.Flags().Changed("metadata") {
				entry.Metadata = metadata
			}

			switch entry.Type {
			case model.EntryTypeCredentials:
				if err := updateCredentials(cmd, vaultManager, &entry, loginName, password); err != nil {
					return err
				}
			case model.EntryTypeText:
				if err := updateText(cmd, vaultManager, &entry, text); err != nil {
					return err
				}
			case model.EntryTypeCard:
				if err := updateCard(cmd, vaultManager, &entry, number, holder, expiry, cvv); err != nil {
					return err
				}
			case model.EntryTypeBinary:
				if err := updateBinary(cmd, vaultManager, &entry, filePath); err != nil {
					return err
				}
			default:
				return fmt.Errorf("неподдерживаемый тип записи: %s", entry.Type)
			}

			if err := client.UpdateEntry(context.Background(), &entry); err != nil {
				return err
			}

			localStorage.UpsertEntry(entry)
			if err := localStorage.Save(); err != nil {
				return err
			}

			printSuccess("Запись %s обновлена (version: %d)", entry.ID, entry.Version)
			return nil
		},
	}

	cmd.Flags().StringVar(&metadata, "metadata", "", "entry metadata")
	cmd.Flags().StringVar(&loginName, "login", "", "stored login (credentials)")
	cmd.Flags().StringVar(&password, "password", "", "stored password (credentials)")
	cmd.Flags().StringVar(&text, "text", "", "text content (text)")
	cmd.Flags().StringVar(&number, "number", "", "card number (card)")
	cmd.Flags().StringVar(&holder, "holder", "", "card holder (card)")
	cmd.Flags().StringVar(&expiry, "expiry", "", "expiry date (card)")
	cmd.Flags().StringVar(&cvv, "cvv", "", "cvv (card)")
	cmd.Flags().StringVar(&filePath, "file", "", "path to binary file (binary)")
	return cmd
}

func hasUpdateFlags(cmd *cobra.Command) bool {
	flags := []string{"metadata", "login", "password", "text", "number", "holder", "expiry", "cvv", "file"}
	for _, name := range flags {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

func loadEntry(localStorage *storage.LocalStorage, client *api.Client, entryID uuid.UUID) (model.Entry, error) {
	if entry, ok := localStorage.GetEntry(entryID); ok {
		return entry, nil
	}

	entry, err := client.GetEntry(context.Background(), entryID)
	if err != nil {
		return model.Entry{}, fmt.Errorf("entry not found, run sync first")
	}

	return *entry, nil
}

func updateCredentials(cmd *cobra.Command, vaultManager *vault.Manager, entry *model.Entry, loginName, password string) error {
	if !cmd.Flags().Changed("login") && !cmd.Flags().Changed("password") {
		return nil
	}

	var payload model.CredentialsPayload
	if err := decryptPayload(vaultManager, entry.Payload, &payload); err != nil {
		return err
	}

	if cmd.Flags().Changed("login") {
		payload.Login = loginName
	}
	if cmd.Flags().Changed("password") {
		payload.Password = password
	}

	return encryptEntryPayload(vaultManager, entry, payload)
}

func updateText(cmd *cobra.Command, vaultManager *vault.Manager, entry *model.Entry, text string) error {
	if !cmd.Flags().Changed("text") {
		return nil
	}

	var payload model.TextPayload
	if err := decryptPayload(vaultManager, entry.Payload, &payload); err != nil {
		return err
	}

	payload.Text = text
	return encryptEntryPayload(vaultManager, entry, payload)
}

func updateCard(cmd *cobra.Command, vaultManager *vault.Manager, entry *model.Entry, number, holder, expiry, cvv string) error {
	if !cmd.Flags().Changed("number") && !cmd.Flags().Changed("holder") &&
		!cmd.Flags().Changed("expiry") && !cmd.Flags().Changed("cvv") {
		return nil
	}

	var payload model.CardPayload
	if err := decryptPayload(vaultManager, entry.Payload, &payload); err != nil {
		return err
	}

	if cmd.Flags().Changed("number") {
		payload.Number = number
	}
	if cmd.Flags().Changed("holder") {
		payload.Holder = holder
	}
	if cmd.Flags().Changed("expiry") {
		payload.ExpiryDate = expiry
	}
	if cmd.Flags().Changed("cvv") {
		payload.CVV = cvv
	}

	return encryptEntryPayload(vaultManager, entry, payload)
}

func updateBinary(cmd *cobra.Command, vaultManager *vault.Manager, entry *model.Entry, filePath string) error {
	if !cmd.Flags().Changed("file") {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	return encryptEntryPayload(vaultManager, entry, model.BinaryPayload{Data: data})
}

func encryptEntryPayload(vaultManager *vault.Manager, entry *model.Entry, payload any) error {
	encrypted, err := vaultManager.EncryptPayload(payload)
	if err != nil {
		return err
	}

	entry.Payload = encrypted
	return nil
}
