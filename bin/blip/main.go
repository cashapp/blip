package main

import (
	"log"

	"github.com/square/blip/server"
)

func main() {
	s := server.Server{}
	if err := s.Boot(); err != nil {
		log.Fatalf("server.Boot failed: %s", err)
	}

	if err := s.Run(); err != nil {
		log.Fatalf("server.Run failed: %s", err)
	} else {
		log.Println("clean shutdown")
	}
}
