package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type RPCResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func handleRPCCommand(args []string) {
	if len(args) == 0 {
		printJSONError("missing rpc subcommand (search|episodes|resolve|play|health)")
		os.Exit(1)
	}

	subcmd := args[0]
	resolver := NewMultiProviderResolver()

	switch subcmd {
	case "search":
		if len(args) < 2 {
			printJSONError("usage: clare rpc search <query>")
			os.Exit(1)
		}
		shows, err := resolver.Search(args[1], "sub")
		if err != nil {
			printJSONError(err.Error())
			os.Exit(1)
		}
		printJSONSuccess(shows)

	case "resolve":
		if len(args) < 3 {
			printJSONError("usage: clare rpc resolve <query> <epNo>")
			os.Exit(1)
		}
		show, stream, err := resolver.ResolveWithFallback(args[1], "sub", args[2], "best")
		if err != nil {
			printJSONError(err.Error())
			os.Exit(1)
		}
		printJSONSuccess(map[string]any{
			"show":   show,
			"stream": stream,
		})

	case "health":
		summary, err := ValidateSessionTrace("")
		if err != nil {
			printJSONError(err.Error())
			os.Exit(1)
		}
		printJSONSuccess(summary)

	default:
		printJSONError(fmt.Sprintf("unknown rpc subcommand: %s", subcmd))
		os.Exit(1)
	}
}

func printJSONSuccess(data any) {
	resp := RPCResponse{Success: true, Data: data}
	b, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(b))
}

func printJSONError(msg string) {
	resp := RPCResponse{Success: false, Error: msg}
	b, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(b))
}
