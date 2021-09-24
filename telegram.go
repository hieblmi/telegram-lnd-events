package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

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
		Name:     "BohemianRhapsodyObserver",
		tgBot:    TgBot,
		tgChatId: config.TgChatId,
	})
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
	html := fmt.Sprintf("<b>New %s</b>\n", e.Type)
	html += fmt.Sprintf("From: %s\n", e.FromAlias)
	html += fmt.Sprintf("To: %s\n", e.ToAlias)
	switch e.Type {
	case "ForwardEvent":
		{
			html += fmt.Sprintf("%d sats --> %d sats\n", e.IncomingMSats/1000, e.OutgoingMSats/1000)
			html += fmt.Sprintf("Fee: %d sats\n", (e.IncomingMSats-e.OutgoingMSats)/1000)
		}
	}

	log.Println(html)

	message := tgbotapi.NewMessage(t.tgChatId, html)
	message.ParseMode = tgbotapi.ModeHTML
	return message
}
