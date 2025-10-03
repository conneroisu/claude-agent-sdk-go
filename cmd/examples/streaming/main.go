package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
)

func main() {
	ctx := context.Background()

	client := claude.NewClient(
		&options.AgentOptions{
			MaxTurns: intPtr(10),
		},
		nil,
		nil,
	)

	if err := client.Connect(ctx, nil); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing client: %v", err)
		}
	}()

	fmt.Println("Connected to Claude. Type 'exit' to quit.")
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\nYou: ")
		if !scanner.Scan() {
			break
		}

		userInput := scanner.Text()
		if userInput == "exit" {
			break
		}

		if err := client.SendMessage(ctx, userInput); err != nil {
			log.Printf("Error sending message: %v", err)

			continue
		}

		msgCh, errCh := client.ReceiveMessages(ctx)

		for {
			select {
			case msg, ok := <-msgCh:
				if !ok {
					goto NextInput
				}

				switch m := msg.(type) {
				case *messages.AssistantMessage:
					fmt.Println("\nClaude:")
					for _, block := range m.Content {
						if textBlock, ok := block.(messages.TextBlock); ok {
							fmt.Printf("  %s\n", textBlock.Text)
						}
					}

				case *messages.ResultMessageSuccess:
					fmt.Printf(
						"\n[Turn completed in %dms]\n",
						m.DurationMs,
					)

					goto NextInput
				}

			case err := <-errCh:
				if err != nil {
					log.Printf("Error: %v", err)
				}

				goto NextInput
			}
		}

	NextInput:
	}

	fmt.Println("\nGoodbye!")
}

func intPtr(i int) *int {
	return &i
}
