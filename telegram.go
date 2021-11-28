package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/hieblmi/go-lnd-router-events/events"
)

type Config struct {
	MacaroonPath  string
	CertPath      string
	RpcHost       string
	TelegramToken string
	TgChatId      int64
}

type LndEventObserver struct {
	Name     string
	tgBot    *tgbotapi.BotAPI
	tgChatId int64
}

var dollarSign string

func main() {
	c := flag.String("config", "./config.json", "Specify the configuration file")
	flag.Parse()
	file, err := os.Open(*c)
	if err != nil {
		log.Fatal("Cannot open config file: ", err)
	}
	defer file.Close()

	config := Config{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal("Cannot decode config JSON: ", err)
	}

	b, err := json.MarshalIndent(config, "", "      ")
	if err != nil {
		log.Println("Cannot indent json config.")
	}
	log.Printf("Printing config.json: %s\n", string(b))

	// parse dollar Unicode sign
	h := "0x0001F4B2"
	i, _ := strconv.ParseInt(h, 0, 64)
	dollarSign = html.UnescapeString(string(i))

	// intialize Telegram Bot
	TgBot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		log.Panic(err)
	}

	// start lnd event listener
	listener := events.New(&events.Config{
		MacaroonPath: config.MacaroonPath,
		CertPath:     config.CertPath,
		RpcHost:      config.RpcHost,
	})

	listener.Register(&LndEventObserver{
		Name:     "MyObserver",
		tgBot:    TgBot,
		tgChatId: config.TgChatId,
	}, events.Forward)

	listener.Start()
}

func (t *LndEventObserver) GetName() string {
	return t.Name
}

func (t *LndEventObserver) Update(e *events.Event) {
	_, err := t.tgBot.Send(t.constructTelegramMessage(e))
	if err != nil {
		log.Fatal(err)
	}
}

func (t *LndEventObserver) constructTelegramMessage(e *events.Event) tgbotapi.MessageConfig {
	html := fmt.Sprintf("New <b>%s</b> (in msat)\n", e.Type)
	switch e.Type {
	case events.Forward:
		{
			html += fmt.Sprintf("<b>%s</b>(%d)\n", e.FromAlias, e.IncomingMSats)
			html += fmt.Sprintf("\tTO \n")
			html += fmt.Sprintf("<b>%s</b>(%d)\n", e.ToAlias, e.OutgoingMSats)
			html += fmt.Sprintf("%sEarned: %d\n", dollarSign, (e.IncomingMSats - e.OutgoingMSats))
		}
	}

	log.Println(html)

	message := tgbotapi.NewMessage(t.tgChatId, html)
	message.ParseMode = tgbotapi.ModeHTML
	return message
}
