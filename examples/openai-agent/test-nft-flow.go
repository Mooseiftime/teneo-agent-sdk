package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Teneo-Protocol/teneo-agent-sdk/pkg/agent"
	"github.com/joho/godotenv"
)

// This test file helps verify that NFT_TOKEN_ID is being read correctly
func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Display what we're loading
	fmt.Println("=== NFT Token ID Flow Test ===")
	fmt.Println()

	privateKey := os.Getenv("PRIVATE_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	nftTokenID := os.Getenv("NFT_TOKEN_ID")

	fmt.Printf("PRIVATE_KEY: %s\n", maskString(privateKey))
	fmt.Printf("OPENAI_API_KEY: %s\n", maskString(openaiKey))
	fmt.Printf("NFT_TOKEN_ID: %s\n", nftTokenID)
	fmt.Println()

	if privateKey == "" {
		log.Fatal("❌ PRIVATE_KEY not set")
	}
	if openaiKey == "" {
		log.Fatal("❌ OPENAI_API_KEY not set")
	}

	fmt.Println("Creating OpenAI agent with simple config...")
	fmt.Println("(Watch for NFT-related log messages)")
	fmt.Println()

	// Create agent - should log what it's doing with NFT
	simpleAgent, err := agent.NewSimpleOpenAIAgent(&agent.SimpleOpenAIAgentConfig{
		PrivateKey: privateKey,
		OpenAIKey:  openaiKey,
		// Not setting TokenID or Mint - let it auto-detect from env
	})

	if err != nil {
		log.Fatalf("❌ Failed to create agent: %v", err)
	}

	fmt.Println()
	fmt.Println("✅ Agent created successfully!")
	fmt.Println("Check the logs above to see NFT Token ID handling")

	// Don't run the agent, just test the creation
	_ = simpleAgent
}

func maskString(s string) string {
	if s == "" {
		return "<not set>"
	}
	if len(s) <= 10 {
		return "****"
	}
	return s[:6] + "..." + s[len(s)-4:]
}
