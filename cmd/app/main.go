package main

import (
	"WhatsappSender/internal/telegram"
	"WhatsappSender/pkg/config"
	"context"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

)

const configPath = "config.yml"

func main() {

	config, err := config.New(configPath)
	if err != nil {
		log.Fatal(err)
	}

	bot := telegram.New(config)
	bot.Start()
	
	}

func WAConnect() (*whatsmeow.Client, error) {
	container, err := sqlstore.New("sqlite3", "file:wapp.db?_foreign_keys=on", waLog.Noop)
	if err != nil {
		return nil, err
	}
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	client := whatsmeow.NewClient(deviceStore, waLog.Noop)
	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			return nil, err
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println(evt.Code)
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				qrc, err := qrcode.New(evt.Code)
				if err != nil {
					fmt.Printf("could not generate QRCode: %v", err)
					return nil, err
				}
				w, err := standard.New("repo-qrcode.jpeg")
				if err != nil {
					fmt.Printf("standard.New failed: %v", err)
					return nil, err
				}
				
				// save file
				if err = qrc.Save(w); err != nil {
					fmt.Printf("could not save image: %v", err)
				}
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err := client.Connect()
		if err != nil {
			return nil, err
		}
		fmt.Println("Connected")
	}
	return client, nil
}

