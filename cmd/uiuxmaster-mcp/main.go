package main

import (
	"context"
	"log"

	"github.com/Homiakus/UiUxMaster/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcpserver.New(mcpserver.Config{})
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
