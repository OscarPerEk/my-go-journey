package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func get_cubes(text string, color string) int {
	color_pattern := regexp.MustCompile(` (\d+) ` + color)
	color_match := color_pattern.FindStringSubmatch(text)
	if len(color_match) == 0 {
		return 0
	}
	cubes, _ := strconv.Atoi(color_match[1])
	return cubes
}

func get_id(text string, req_green int, req_blue int, req_red int) (int, error) {
	id_pattern := regexp.MustCompile(`Game (\d+):`)
	id := id_pattern.FindStringSubmatch(text)
	id_int, _ := strconv.Atoi(id[1])
	for i, sub := range strings.SplitAfter(text, ";") {
		fmt.Printf("i = %v, char = %v\n", i, sub)
		if !(get_cubes(sub, "green") <= req_green && get_cubes(sub, "blue") <= req_blue && get_cubes(sub, "red") <= req_red) {
			return 0, errors.New("Too many qubes")
		}
	}
	return id_int, nil
}

func main() {
	file, err := os.Open("input.txt")
	if err != nil {
		log.Fatal(err)
	}
	sum := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var text string = scanner.Text()
		id, err := get_id(
			text,
			13,
			14,
			12,
		)
		if err == nil {
			sum += id
		}
	}
	fmt.Printf("The sum is: %d", sum)
}
