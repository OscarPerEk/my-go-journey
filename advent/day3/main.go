package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Symbol struct {
	symbol        string
	y_coordinate  int
	x_coordinates int
}

type Number struct {
	number        int
	y_coordinate  int
	x_coordinates []int
}

func is_adjacent(number Number, symbols []Symbol) bool {
	for _, symbol := range symbols {
		if symbol.y_coordinate-1 == number.y_coordinate || symbol.y_coordinate+1 == number.y_coordinate || symbol.y_coordinate == number.y_coordinate {
			for _, x := range number.x_coordinates {
				if symbol.x_coordinates-1 == x || symbol.x_coordinates+1 == x || symbol.x_coordinates == x {
					return true
				}
			}
		}
	}
	return false
}

func generate_range(start int, end int) []int {
	var results []int
	for i := start; i <= end; i++ {
		results = append(results, i)
	}
	return results
}

func parse_line(text string, row int, numbers []Number, symbols []Symbol) ([]Number, []Symbol) {
	i := 0
	for _, sub := range strings.Split(text, ".") {
		fmt.Println(sub)
		fmt.Println(i)
		numb, err := strconv.Atoi(sub)
		if err == nil {
			numbers = append(
				numbers,
				Number{
					number:        numb,
					y_coordinate:  row,
					x_coordinates: generate_range(i, i+len(sub)-1),
				},
			)
		} else if sub == "" {
			i += 1
			continue
		} else {
			symbols = append(
				symbols,
				Symbol{
					symbol:        sub,
					y_coordinate:  row,
					x_coordinates: i,
				},
			)
		}
		i += utf8.RuneCountInString(sub) + 1
	}
	return numbers, symbols
}

func main() {
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
		numbers, symbols = parse_line(text, row, numbers, symbols)
		row += 1
	}
	fmt.Printf("numbers: %+v\n", numbers)
	fmt.Printf("symbols: %+v\n", symbols)
	sum := 0
	for _, number := range numbers {
		if is_adjacent(number, symbols) {
			sum += number.number
		} else {
		}
	}
	fmt.Printf("The sum is: %d", sum)
}
