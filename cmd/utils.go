package cmd

import "strings"

func processListArgs(args []string) []string {
	var listArgs []string
	for _, arg := range args {
		argList := strings.Split(arg, ",")
		for _, item := range argList {
			item := strings.TrimSpace(item)
			if len(item) > 0 {
				listArgs = append(listArgs, item)
			}
		}
	}
	return listArgs
}
