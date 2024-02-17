package main

import (
	"fmt"
	"log"

	"my-go-journey/package1"
)

func main() {
	log.SetPrefix("my logger prefix: ")
	log.SetFlags(0)
	message, err := package1.MyHelloFunction("oscar")
	if err != nil {
		log.Fatal("message is empty")
	}
	fmt.Println(message)
	messages := package1.MapHelloFunction([]string{"name1", "name2", "name3"})
	fmt.Println(messages)
}
