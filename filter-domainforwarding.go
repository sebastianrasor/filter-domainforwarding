package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"log"
)

var version string

var outputChannel chan string

var filters = map[string]func(string, []string){
	"rcpt-to": filterRcptTo,
}

func produceOutput(msgType string, sessionId string, token string, format string, parameter ...string) {
	var out string

	if version < "0.5" {
		out = msgType + "|" + token + "|" + sessionId
	} else {
		out = msgType + "|" + sessionId + "|" + token
	}
	out += "|" + fmt.Sprintf(format)
	for k := range parameter {
		out += "|" + fmt.Sprintf(parameter[k])
	}

	outputChannel <- out
}

func filterRcptTo(sessionId string, params []string) {
	token := params[0]
	recipient := params[1]

	parts := strings.Split(recipient, "@")
	if len(parts) == 1 {
		produceOutput("filter-result", sessionId, token, "proceed")
		return
	}

	if parts[1] == "hmail.app" && strings.Contains(parts[0], "_") {
		produceOutput("filter-result", sessionId, token, "rewrite", "<" + strings.Replace(parts[0], "_", "@", 1) + ">")
		return
	} else {
		produceOutput("filter-result", sessionId, token, "proceed")
		return
	}
}

func filterInit() {
	for k := range filters {
		fmt.Printf("register|filter|smtp-in|%s\n", k)
	}
	fmt.Println("register|ready")
}

func trigger(currentSlice map[string]func(string, []string), atoms []string) {
	if handler, ok := currentSlice[atoms[4]]; ok {
		handler(atoms[5], atoms[6:])
	} else {
		log.Fatalf("invalid phase: %s", atoms[4])
	}
}

func skipConfig(scanner *bufio.Scanner) {
	for {
		if !scanner.Scan() {
			os.Exit(0)
		}
		line := scanner.Text()
		if line == "config|ready" {
			return
		}
	}
}

func main() {
	flag.Parse()
	scanner := bufio.NewScanner(os.Stdin)
	skipConfig(scanner)
	filterInit()

	outputChannel = make(chan string)
	go func() {
		for line := range outputChannel {
			fmt.Println(line)
		}
	}()

	for {
		if !scanner.Scan() {
			os.Exit(0)
		}

		line := scanner.Text()
		atoms := strings.Split(line, "|")
		if len(atoms) < 6 {
			log.Fatalf("missing atoms: %s", line)
		}

		version = atoms[1]

		if atoms[0] != "filter" {
			log.Fatalf("invalid stream: %s", atoms[0])
		}

		trigger(filters, atoms)
	}
}
