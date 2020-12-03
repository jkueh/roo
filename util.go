package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func getStringInputFromUser(prompt string) string {
	fmt.Print(prompt + ": ")
	text, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		log.Fatalln("An error occurred while trying to get user input:", err)
	}
	return strings.TrimSuffix(text, "\n")
}

func oneTimePasscodeIsValid(code string) (bool, error) {
	// Step 0 - It must be at least 6 characters
	if len(code) < 6 {
		return false, fmt.Errorf("Code provided was less than 6 characters long")
	}
	_, err := strconv.Atoi(code)
	if err != nil {
		return false, err
	}
	return true, nil
}
