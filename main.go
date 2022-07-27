package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/pelletier/go-toml/v2"
)

var g 		*gateway.Gateway
var client 	*api.Client
var ctx 	context.Context
var me 		discord.User

var messageBuffer string

var cfg struct {
	Token	string
	GuildID	discord.GuildID
	ChannelID discord.ChannelID
}

var messages chan string

func main() {
	// Get the config file
	configFile, err := os.ReadFile("config.toml")
	if(err != nil) {log.Fatalln(err)}

	// Unmarshal the contents into the global config object
	err = toml.Unmarshal([]byte(configFile),&cfg)
	if(err != nil) {log.Fatalln(err)}

	// Initialize the client
	client = api.NewClient("Bot "+cfg.Token)
	me_, err := client.Me() 
	if(err != nil) {
		fmt.Println(err)
		return
	}
	me = *me_

	// Start listening for events.
	ctx, _ = signal.NotifyContext(context.Background(), os.Interrupt)
	g, err = gateway.NewWithIntents(ctx, cfg.Token, gateway.IntentGuilds, gateway.IntentGuildMessages)
	if err != nil {
		log.Fatalln("failed to initialize gateway:", err)
	}

	// set up a go routine for detecting when the program is closed.
	watchForClose()

	// set up a go routine for detecting when something is typed in the relevant channel
	go watchForChannelMessages()

	// set up a go routine for watching the message buffer and posting it when it's not empty
	go watchMessageBuffer()

	// listen to the stdin
	watchForStdin()
}

func watchForClose() {
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT )
    go func() {
	    <-sigs
	    Print("Program stopped.")
	    os.Exit(0)
    }()
}

func watchForChannelMessages() {
	for op := range g.Connect(ctx) {
		switch data := op.Data.(type) {
		case *gateway.ReadyEvent:
			fmt.Println("Online.")
		case *gateway.MessageCreateEvent:
			if(data.ChannelID != cfg.ChannelID) {continue}
			if(data.Author == me) {continue}
			os.Stdout.Write([]byte(data.Content))
			os.Stdout.Write([]byte("\n"))
		}
	}
}

func watchForStdin() {
	// create a new stdin reader
	reader := bufio.NewReader(os.Stdin)
	// And start reading from it.
	for {
		message, _ := reader.ReadString('\n')
		if(message == "") {
			message = "­"
		}
		switch message {
			// By default, just send the output to a channel.
			default: 
				messageBuffer += message
		}
	}
}

func watchMessageBuffer() {
	for {
		time.Sleep(time.Second * 1)
		if(messageBuffer != "") {
			fmt.Println(messageBuffer)
			Print(messageBuffer)
			messageBuffer = ""
		}
	}
}

func Print(str string) {
	_, err := client.SendMessage(cfg.ChannelID,str)
	if(err != nil) {
		fmt.Println(err)
	}
}