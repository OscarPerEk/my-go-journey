package package1

import (
	"errors"
	"fmt"
	"math/rand"
)

func MyHelloFunction(name string) (string, error) {
	if name == "" {
		return "", errors.New("my error message")
	}
	my_slice := []string{
		"hello %s",
		"hi %s",
		"hey %s",
	}
	return fmt.Sprintf(my_slice[rand.Intn(len(my_slice))], name), nil
}

func MapHelloFunction(names []string) map[string]string {
	my_map := make(map[string]string)
	for i, name := range names {
		fmt.Println("looping through names:")
		fmt.Println(i, name)
		mess, err := MyHelloFunction(name)
		fmt.Printf("adding mess %v", mess)
		fmt.Printf("adding err %v", err)
		if err == nil {
			fmt.Printf("adding mess %v", mess)
			my_map[name] = mess
		}
	}
	return my_map
}
