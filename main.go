package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/pelletier/go-toml/v2"
)

var client 		*api.Client


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

	fmt.Println("Online.")

	// set up a go routine for detecting when the program is closed.
	watchForClose()

	// listen to the stdin
	watchForStdin()
}

func watchForClose() {
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT )
    go func() {
	    wow := <-sigs
	    fmt.Println(wow)
	    Print("Program stopped.")
	    os.Exit(0)
    }()
   

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
				Print(message)
		}
	}
}

func Print(str string) {
	_, err := client.SendMessage(cfg.ChannelID,str)
	if(err != nil) {
		fmt.Println(err)
	}
}