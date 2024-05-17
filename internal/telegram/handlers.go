package telegram

import (
	"WhatsappSender/internal/localstorage"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"github.com/xuri/excelize/v2"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleCommandStart(update *tgbotapi.Update, userID int64) {
	b.LocalStorage.SetState(userID, localstorage.StateIdle)
	wac, qrChan, err := WAConnect(b.TgAPI, userID)
	if err != nil {
		b.SendTo(userID, "Ошибка при авторазиции", nil)
		fmt.Println(err)
		return
	}
	b.LocalStorage.SetState(userID, localstorage.StateWaitingScanQr)
	for evt := range qrChan {
		if evt.Event == "code" {
			_, ok := b.LocalStorage.Get(userID)
			if !ok {
				return
			}
			qrc, err := qrcode.New(evt.Code)
			if err != nil {
				b.SendTo(userID, "Ошибка при авторазиции", nil)
			}
			pipeReader, pipeWriter := io.Pipe()
			fileReader := tgbotapi.FileReader{Name: "qr.jpg", Reader: pipeReader}	
			writer := standard.NewWithWriter(pipeWriter)
			go qrc.Save(writer)
			
			photo := tgbotapi.NewPhoto(userID, fileReader)
			photo.Caption = "Отсканируйте этот qr-код"

			b.TgAPI.Send(photo)
			
		} else {
			if !b.LocalStorage.SetWAClient(userID, wac) {
				b.SendTo(userID, "Неизвестная ошибка", nil)
				return
			}
			b.LocalStorage.SetState(userID, localstorage.StateWaitingFile)
			msg := tgbotapi.NewMessage(userID, "Авторизация прошла успешно.\n" + 
								"Теперь отправьте excel файл с номерами")	
			b.TgAPI.Send(msg)
		}
	}

}

func WAConnect(tgapi *tgbotapi.BotAPI, userID int64) (*whatsmeow.Client, <-chan whatsmeow.QRChannelItem, error) {
	container, err := sqlstore.New("sqlite3", "file:wapp.db?_foreign_keys=on", waLog.Noop)
	if err != nil {
		return nil, nil, err
	}
	deviceStore := container.NewDevice()
	if err != nil {
		return nil, nil, err
	}
	store.DeviceProps.Os = proto.String("Firefox (Windows)")
	client := whatsmeow.NewClient(deviceStore, waLog.Noop)
	qrChan, _ := client.GetQRChannel(context.Background())
	err = client.Connect()
	if err != nil {
		return nil, nil, err
	}
	return client, qrChan, nil
}

func (b *Bot) handleDocument(update *tgbotapi.Update, userID int64) {
	document := update.Message.Document
	if document.MimeType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		if !b.LocalStorage.SetFileID(userID, document.FileID)	{
			b.SendTo(userID, "Неизвестная ошибка", nil)
			return
		}
		b.LocalStorage.SetState(userID, localstorage.StateWaitingInterval)
		b.SendTo(userID, "Теперь введите интервал между сообщениями(в секундах)", nil)
	} else {
		b.SendTo(userID, "Неверный формат файла. Попробуйте ещё раз", nil)
	}
}

func (b *Bot) handleTextToSend(update *tgbotapi.Update, userID int64, userInfo *localstorage.UserInfo) {
	b.LocalStorage.SetState(userID, localstorage.StateIdle)
	fileURL, err := b.TgAPI.GetFileDirectURL(userInfo.SentedFileID)
	if err != nil {
		b.SendTo(userID, "Неизвестная ошибка\nначните заново", nil)
		return
	}

	resp, err := http.Get(fileURL)
	if err != nil {
		b.SendTo(userID, "Неизвестная ошибка\nначните заново", nil)
		return
	}
	defer resp.Body.Close()
	f, err := excelize.OpenReader(resp.Body)
	if err != nil {
		b.SendTo(userID, "Ошибка при чтении файла\nначните заново", nil)
		return
	}

	cellB1, err := f.GetCellValue("Лист1", "B1")
	if err != nil {
		b.SendTo(userID, "Неизвестная ошибка\nначните заново", nil)
		return
	}
	
	phoneNumbers := []string{}
	if cellB1 == "Телефон" {
		_, ok := b.LocalStorage.Get(userID)
			if !ok {
				return
			}
		for i := 2; ;i++ {
			cell, err := f.GetCellValue("Лист1", "B"+strconv.Itoa(i))
			if err != nil {
				fmt.Println(err)
				break
			}
			if cell == "" {
				b.SendTo(userID, fmt.Sprintf("Загружено %d номеров\nНачинаю рассылку", len(phoneNumbers)), nil)
				break
			}
			phoneNumbers = append(phoneNumbers, cell)
		}
	}

	wac := userInfo.WAClient

	succsessMsgsCount := 0
	for i, phone := range phoneNumbers {
		if i > 0 {
			time.Sleep(time.Duration(userInfo.MsgInterval) * time.Second)
		}
		_, err = wac.SendMessage(context.Background(), types.JID{
				User:   phone,
				Server: types.DefaultUserServer,
			}, &waProto.Message{
				Conversation: proto.String(update.Message.Text),
			})
			if err != nil {
				b.SendTo(userID, fmt.Sprintf("Ошибка при отправке сообщения на номер %s.", phone), nil)
			}
			b.SendTo(userID, fmt.Sprintf("Сообщение на номер %s отправлено.", phone), nil)
			succsessMsgsCount++
	}

	b.SendTo(userID, fmt.Sprintf("Сообщение отправлено на %d из %d номеров🎉🎉🎉", succsessMsgsCount, len(phoneNumbers)), nil)

	wac.Logout()
}

func (b *Bot) handleCommandStop(update *tgbotapi.Update, userID int64) {
	b.LocalStorage.Delete(userID)
	b.SendTo(userID, "Бот останавливается, пожалуйста подождите 1-2 минуты, прежде чем начать работу снова", nil)
}

func (b *Bot) handleGetInterval(update *tgbotapi.Update, userID int64) {
	interval, err := strconv.Atoi(update.Message.Text)
	if err != nil {
		b.SendTo(userID, "Неверный интервал.\nПопробуйте снова", nil)
	}
	b.LocalStorage.SetInterval(userID, interval)
	b.LocalStorage.SetState(userID, localstorage.StateWaitingText)
	b.SendTo(userID, "Теперь введите текст для отправки", nil)
}
