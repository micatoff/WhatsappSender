package telegram

import (
	"WhatsappSender/internal/localstorage"
	"WhatsappSender/pkg/config"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	Config         *config.Config
	TgAPI          *tgbotapi.BotAPI
	LocalStorage   *localstorage.Storage
}

func New(c *config.Config) *Bot {
	return &Bot{Config: c, LocalStorage: localstorage.New()}
}

func (b *Bot) handleCommand(update *tgbotapi.Update, userID int64, userInfo *localstorage.UserInfo) {
	switch update.Message.Command() {
	case "start":
		b.handleCommandStart(update, userID)
	case "stop":
		b.handleCommandStop(update, userID)
	}
}

func (b *Bot) handleMessage(update *tgbotapi.Update, userID int64, userInfo *localstorage.UserInfo) {
		if update.Message.Document != nil {
			if userInfo.State == localstorage.StateWaitingFile {
				b.handleDocument(update, userID)
			}
		} else {
			if userInfo.State == localstorage.StateWaitingInterval {
				b.handleGetInterval(update, userID)
			} else if userInfo.State == localstorage.StateWaitingText {
				b.handleTextToSend(update, userID, userInfo)
			}
		}
}


func (b *Bot) Start() {
	bot, err := tgbotapi.NewBotAPI(b.Config.TgApiToken)
	if err != nil {
		log.Panic(err)
	}

	b.TgAPI = bot

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			userID := update.Message.From.ID
			userInfo, ok := b.LocalStorage.Get(userID)
			if !ok {
				b.LocalStorage.SetState(userID, localstorage.StateIdle)
			}
			if update.Message.IsCommand() {
				go b.handleCommand(&update, userID, userInfo)
			} else {
				go b.handleMessage(&update, userID, userInfo)
			}
		}
	}
}

func (b *Bot) SendTo(userID int64, text string, replyMarkup interface{}) {
	msg := tgbotapi.NewMessage(userID, text)
	msg.ReplyMarkup = replyMarkup
	b.TgAPI.Send(msg)
}

