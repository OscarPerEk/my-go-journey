package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
)

func get_values(text string) (int, error) {
	digits := [10]rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
	var first int
	var second int
	found := false
	out := false
	for i, char := range text {
		x := char - '0'
		fmt.Printf("indx = %v\n", i)
		fmt.Printf("Char = %v\n", char)
		fmt.Printf("Int(x) = %v\n", int(x))
		fmt.Printf("Int(char-'0') = %v\n", int(char-'0'))
		for _, digit := range digits {
			if char == digit {
				first = int(digit-'0') * 10
				out = true
				break
			}
		}
		if out {
			break
		}
	}
	out = false
	for i := len(text) - 1; i >= 0; i-- {
		for _, dig := range digits {
			if dig == rune(text[i]) {
				second = int(rune(text[i]) - '0')
				fmt.Printf("second int -0 = %v\n", int(second))
				x := second - '0'
				fmt.Printf("indx = %v\n", i)
				fmt.Printf("Char = %v\n", text[i])
				fmt.Printf("Int(x) = %v\n", int(x))
				fmt.Printf("Int(dig-'0') = %v\n", int(dig-'0'))
				found = true
				out = true
				break
			}
		}
		if out {
			break
		}
	}
	if !found {
		return 0, errors.New("No digits found")
	}
	//concatenate the strings first and second
	fmt.Printf("first = %v. last = %v", first, second)
	concat := first + second
	fmt.Printf("concat = %v", concat)
	return concat, nil
}

func main() {
	file, err := os.Open("input.txt")
	if err != nil {
		log.Fatal(err)
	}
	//read line by line from a txt file
	scanner := bufio.NewScanner(file)
	sum := 0
	for scanner.Scan() {
		var text string = scanner.Text()
		values, err := get_values(text)
		fmt.Println(values)
		if err == nil {
			sum += values
		}
	}
	fmt.Printf("The sum is: %d", sum)
}
