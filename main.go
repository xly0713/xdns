package main

import (
	"log"
	"os"
	"os/signal"
)

func main() {
	srv := &server{
		host: "",
		port: 53,
	}

	srv.run()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	sig := <-c
	log.Printf("signal %s received, stopping", sig)
}
