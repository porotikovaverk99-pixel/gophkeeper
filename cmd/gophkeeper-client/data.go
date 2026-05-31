package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
)

func addLoginCmd() *cobra.Command {
	var metadata, loginName, password string

	cmd := &cobra.Command{
		Use:   "add-login",
		Short: "Add login/password entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if loginName == "" || password == "" {
				return fmt.Errorf("укажите --login и --password")
			}

			localStorage, client, vaultManager, err := openVault()
			if err != nil {
				return err
			}

			entry, err := vaultManager.BuildEntry(model.EntryTypeCredentials, metadata, model.CredentialsPayload{
				Login:    loginName,
				Password: password,
			})
			if err != nil {
				return err
			}

			if err := client.CreateEntry(context.Background(), entry); err != nil {
				return err
			}

			localStorage.UpsertEntry(*entry)
			if err := localStorage.Save(); err != nil {
				return err
			}

			printSuccess("Логин/пароль сохранён (id: %s)", entry.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&metadata, "metadata", "", "entry metadata")
	cmd.Flags().StringVar(&loginName, "login", "", "stored login")
	cmd.Flags().StringVar(&password, "password", "", "stored password")
	return cmd
}

func addTextCmd() *cobra.Command {
	var metadata, text string

	cmd := &cobra.Command{
		Use:   "add-text",
		Short: "Add text entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if text == "" {
				return fmt.Errorf("укажите --text")
			}

			localStorage, client, vaultManager, err := openVault()
			if err != nil {
				return err
			}

			entry, err := vaultManager.BuildEntry(model.EntryTypeText, metadata, model.TextPayload{Text: text})
			if err != nil {
				return err
			}

			if err := client.CreateEntry(context.Background(), entry); err != nil {
				return err
			}

			localStorage.UpsertEntry(*entry)
			if err := localStorage.Save(); err != nil {
				return err
			}

			printSuccess("Текстовая запись сохранена (id: %s)", entry.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&metadata, "metadata", "", "entry metadata")
	cmd.Flags().StringVar(&text, "text", "", "text content")
	return cmd
}

func addCardCmd() *cobra.Command {
	var metadata, number, holder, expiry, cvv string

	cmd := &cobra.Command{
		Use:   "add-card",
		Short: "Add bank card entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if number == "" || holder == "" || expiry == "" || cvv == "" {
				return fmt.Errorf("укажите --number, --holder, --expiry и --cvv")
			}

			localStorage, client, vaultManager, err := openVault()
			if err != nil {
				return err
			}

			entry, err := vaultManager.BuildEntry(model.EntryTypeCard, metadata, model.CardPayload{
				Number:     number,
				Holder:     holder,
				ExpiryDate: expiry,
				CVV:        cvv,
			})
			if err != nil {
				return err
			}

			if err := client.CreateEntry(context.Background(), entry); err != nil {
				return err
			}

			localStorage.UpsertEntry(*entry)
			if err := localStorage.Save(); err != nil {
				return err
			}

			printSuccess("Карта сохранена (id: %s)", entry.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&metadata, "metadata", "", "entry metadata")
	cmd.Flags().StringVar(&number, "number", "", "card number")
	cmd.Flags().StringVar(&holder, "holder", "", "card holder")
	cmd.Flags().StringVar(&expiry, "expiry", "", "expiry date")
	cmd.Flags().StringVar(&cvv, "cvv", "", "cvv")
	return cmd
}

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [id]",
		Short: "Show entry by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entryID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("некорректный id записи")
			}

			localStorage, _, vaultManager, err := openVault()
			if err != nil {
				return err
			}

			entry, ok := localStorage.GetEntry(entryID)
			if !ok {
				return fmt.Errorf("entry not found locally, run sync first")
			}

			switch entry.Type {
			case model.EntryTypeCredentials:
				var payload model.CredentialsPayload
				if err := decryptPayload(vaultManager, entry.Payload, &payload); err != nil {
					return err
				}
				fmt.Printf("type=credentials metadata=%q login=%s password=%s\n", entry.Metadata, payload.Login, payload.Password)
			case model.EntryTypeText:
				var payload model.TextPayload
				if err := decryptPayload(vaultManager, entry.Payload, &payload); err != nil {
					return err
				}
				fmt.Printf("type=text metadata=%q text=%s\n", entry.Metadata, payload.Text)
			case model.EntryTypeCard:
				var payload model.CardPayload
				if err := decryptPayload(vaultManager, entry.Payload, &payload); err != nil {
					return err
				}
				fmt.Printf("type=card metadata=%q number=%s holder=%s expiry=%s cvv=%s\n",
					entry.Metadata, payload.Number, payload.Holder, payload.ExpiryDate, payload.CVV)
			case model.EntryTypeBinary:
				var payload model.BinaryPayload
				if err := decryptPayload(vaultManager, entry.Payload, &payload); err != nil {
					return err
				}
				fmt.Printf("type=binary metadata=%q size=%d bytes\n", entry.Metadata, len(payload.Data))
			default:
				fmt.Printf("type=%s metadata=%q\n", entry.Type, entry.Metadata)
			}

			return nil
		},
	}
}

func deleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete entry by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entryID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("некорректный id записи")
			}

			localStorage, client, _, err := openVault()
			if err != nil {
				return err
			}

			if err := client.DeleteEntry(context.Background(), entryID); err != nil {
				return err
			}

			localStorage.RemoveEntry(entryID)
			if err := localStorage.Save(); err != nil {
				return err
			}

			printSuccess("Запись %s удалена", entryID)
			return nil
		},
	}
}

func addBinaryCmd() *cobra.Command {
	var metadata, filePath string

	cmd := &cobra.Command{
		Use:   "add-binary",
		Short: "Add binary entry from file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("укажите --file")
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}

			localStorage, client, vaultManager, err := openVault()
			if err != nil {
				return err
			}

			entry, err := vaultManager.BuildEntry(model.EntryTypeBinary, metadata, model.BinaryPayload{Data: data})
			if err != nil {
				return err
			}

			if err := client.CreateEntry(context.Background(), entry); err != nil {
				return err
			}

			localStorage.UpsertEntry(*entry)
			if err := localStorage.Save(); err != nil {
				return err
			}

			printSuccess("Файл сохранён (id: %s, %d байт)", entry.ID, len(data))
			return nil
		},
	}

	cmd.Flags().StringVar(&metadata, "metadata", "", "entry metadata")
	cmd.Flags().StringVar(&filePath, "file", "", "path to binary file")
	return cmd
}
