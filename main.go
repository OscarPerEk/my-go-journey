package main

import (
	"fmt"

	"rsc.io/quote/v4"
)

func usePackage() {
	some_var := 10
	fmt.Println(quote.Go(), some_var)
}

func helloWorld() string {
	return "Hello, World!"
}

func printHello(name string, age int) {
	// how to print a string with variables
	fmt.Printf("Hello %s with age %d and hight %v\n", name, age, 2.10)
}

func getHello(name string, age int) string {
	message := fmt.Sprintf("Hello %s with age %v and hight %v", name, age, 2.10)
	return message
}

func main() {
	usePackage()
	printHello("Viktor", 25)
	fmt.Println(helloWorld())
	fmt.Println(getHello("Sophie", 25))
}
