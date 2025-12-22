package utils

import (
	"bufio"
	"log"
	"os"
)

// scan StdIn and send each line to the apiUrl channel

func ScanStdIn(apiUrl chan string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		apiUrl <- scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading stdin: %v\n", err)
	}

	close(apiUrl)
}
