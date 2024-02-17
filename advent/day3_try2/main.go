package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"unicode"
)

type Symbol struct {
	symbol       rune
	y_coordinate int
	x_coordinate int
}

type Number struct {
	number       int
	y_coordinate int
	x_start      int
	x_end        int
}

func get_number(number []rune, row int, i int) Number {
	numb, err := strconv.Atoi(string(number))
	if err != nil {
		log.Fatal(err)
	}
	return Number{
		number:       numb,
		y_coordinate: row,
		x_start:      i - len(number),
		x_end:        i - 1,
	}
}

func parse_line(text string, row int) ([]Number, []Symbol) {
	var numbers []Number
	var symbols []Symbol
	var number []rune = nil
	for i, r := range text {
		if unicode.IsDigit(r) {
			number = append(number, r)
		} else if r == '.' && len(number) > 0 {
			new_number := get_number(number, row, i)
			numbers = append(
				numbers,
				new_number,
			)
			number = nil
		} else if r != '.' {
			symbols = append(
				symbols,
				Symbol{
					symbol:       r,
					y_coordinate: row,
					x_coordinate: i,
				},
			)
			if len(number) > 0 {
				new_number := get_number(number, row, i)
				numbers = append(
					numbers,
					new_number,
				)
				number = nil
			}
		}
	}
	return numbers, symbols
}

func is_adjacent(number Number, symbols []Symbol) bool {
	for _, symbol := range symbols {
		if symbol.y_coordinate-1 <= number.y_coordinate && number.y_coordinate <= symbol.y_coordinate+1 {
			if number.x_start-1 <= symbol.x_coordinate && symbol.x_coordinate <= number.x_end+1 {
				return true
			}
		}
	}
	return false
}

func main() {
	// file, err := os.Open("test.txt")
	file, err := os.Open("input.txt")
	if err != nil {
		log.Fatal((err))
	}
	row := 0
	var numbers []Number
	var symbols []Symbol
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var text string = scanner.Text()
		ns, ss := parse_line(text, row)
		fmt.Printf("Number: %+v\n", ns)
		fmt.Printf("Symbols: %+v\n", ss)
		numbers = append(numbers, ns...)
		symbols = append(symbols, ss...)
		row += 1
	}
	sum := 0
	for _, number := range numbers {
		if is_adjacent(number, symbols) {
			sum += number.number
		}
	}
	fmt.Printf("The sum is: %d", sum)
}
