package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter something: ")
	input, _ := reader.ReadString('
')
	fmt.Printf("You said: %s", strings.TrimSpace(input))
}
