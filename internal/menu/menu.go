// Package menu provides a thin interactive dispatcher over existing commands.
package menu

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Execute func([]string) int

func Run(in io.Reader, out io.Writer, execute Execute) int {
	reader := bufio.NewReader(in)
	for {
		fmt.Fprint(out, "OrangeCTL menu\n1) list  2) validate  3) start  4) stop  5) logs  0) exit\n> ")
		choice, err := reader.ReadString('\n')
		if err != nil {
			return 0
		}
		choice = strings.TrimSpace(choice)
		if choice == "0" || choice == "q" {
			return 0
		}
		commands := map[string]string{"1": "list", "2": "validate", "3": "start", "4": "stop", "5": "logs"}
		command, ok := commands[choice]
		if !ok {
			fmt.Fprintln(out, "Unknown choice")
			continue
		}
		args := []string{command}
		if command == "start" || command == "stop" || command == "logs" {
			fmt.Fprint(out, "Slot: ")
			slot, _ := reader.ReadString('\n')
			args = append(args, strings.TrimSpace(slot))
			if command == "stop" {
				fmt.Fprint(out, "Type yes to confirm: ")
				yes, _ := reader.ReadString('\n')
				if strings.TrimSpace(yes) != "yes" {
					fmt.Fprintln(out, "Cancelled")
					continue
				}
			}
		}
		execute(args)
	}
}
