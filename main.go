package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"

	//"io"
	"log"
	"os"
	"os/exec"
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
var me 		*discord.User

var offset int 	= 0

var messageBuffer *bytes.Buffer = new(bytes.Buffer)

var stdout io.ReadCloser
var stdin io.WriteCloser

var cfg struct {
	Token	string
	GuildID	discord.GuildID
	ChannelID discord.ChannelID
	CommandToPipe string
}

var messages chan []byte
var started chan bool

var err error

func main() {
	// Get the config file
	configFile, err := os.ReadFile("config.toml")
	if(err != nil) {log.Fatalln(err)}

	// Unmarshal the contents into the global config object
	err = toml.Unmarshal([]byte(configFile),&cfg)
	if(err != nil) {log.Fatalln(err)}

	// Initialize the client
	client = api.NewClient("Bot "+cfg.Token)
	me, err = client.Me() 
	if(err != nil) {
		fmt.Println(err)
		return
	}

	// Start listening for events.
	ctx, _ = signal.NotifyContext(context.Background(), os.Interrupt)
	g, err = gateway.NewWithIntents(ctx, cfg.Token, gateway.IntentGuilds, gateway.IntentGuildMessages)
	if err != nil {
		log.Fatalln("failed to initialize gateway:", err)
	}

	// set up a go routine for listing to stdin and sending shit to the program
	go watchForStdin()

	// set up a go routine for detecting when something is typed in the relevant channel
	go watchForStdout()

	// set up a go routine for detecting when the program is closed.
	watchForClose()

	// set up a go routine for watching the message buffer and posting it when it's not empty
	go watchMessageBuffer()

	select {}
}

func watchForClose() {
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT )
    go func() {
	    <-sigs
	    //Print("Program stopped.")
	    os.Exit(0)
    }()
}

func watchForStdout() {
	// for each event sent from the discord bot
	for op := range g.Connect(ctx) {
		switch data := op.Data.(type) {
			case *gateway.ReadyEvent: 			// when the bot is initialized
				fmt.Println("Online.")
				go runCommand()
			case *gateway.MessageCreateEvent: 	// when a message is sent
				// ignore anything not sent in the configured channel
				if(data.ChannelID != cfg.ChannelID) {continue}
				// or by the bot.
				if(data.Author.Username == me.Username) {continue}
				// basically send everything in the message to the terminal.
				for _, v := range data.Content {
					stdin.Write([]byte{byte(v)})
				}
				// and press enter.
				stdin.Write([]byte{'\n'})
		}
	}
}

func watchMessageBuffer() {
	for {
		// every 250 milliseconds...
		time.Sleep(time.Millisecond * 250)
		// get the section of bytes that we haven't read before.
		newBuffer := messageBuffer.Bytes()[offset:]
		if(len(newBuffer) > 0) {
			// send the new text to the discord channel
			Print(string(newBuffer))
			// send it here as well
			fmt.Println(string(newBuffer))
			// update the offset for what we've read
			offset = len(messageBuffer.Bytes())
		}
	}
}

func Print(str string) {
	_, err := client.SendMessage(cfg.ChannelID,str)
	if(err != nil) {
		fmt.Println(err)
	}
}

func runCommand() {
	cmd := exec.Command(cfg.CommandToPipe)

	// errors are ignored because for these to fail is such a rare occurance
	// and if they do fail you have bigger problems; you'll probably see the error
	// long before you see this.
	stdout, _ = cmd.StdoutPipe()
	stdin, _ = cmd.StdinPipe()

	if err := cmd.Start(); err != nil {
		fmt.Println(err)
		return
	}

	messageBuffer.ReadFrom(stdout)
}

func watchForStdin() {
	// create a new stdin reader
	reader := bufio.NewReader(os.Stdin)
	// And start reading from it.
	for {
		message, _ := reader.ReadString('\n')
		for _, v := range message {
			stdin.Write([]byte{byte(v)})
		}
	}
}
