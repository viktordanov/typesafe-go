package typesafe_test

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/hhhapz/typesafe-go"
)

func ExampleClient_SystemOne() {
	client, err := typesafe.New(typesafe.Config{APIKey: flag.Arg(0)})
	if err != nil {
		log.Fatal(err)
	}

	tones := []any{"calm", "annoyed", "angry"}
	result, err := client.SystemOne(context.Background(), typesafe.SystemOneRequest{
		State: "I was charged twice for my order.",
		Questions: typesafe.Questions{
			"billing": typesafe.Noul{Instructions: "Is this message about billing?"},
			"team": typesafe.Choice{
				Instructions: "Which team should handle this message?",
				Criteria:     map[string]any{"billing": nil, "shipping": nil},
			},
			"tone": typesafe.Score{
				Instructions: "How upset is the customer?",
				Criteria:     tones,
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	billing, _ := result.Noul("billing")
	team, _ := result.Choice("team")
	tone, _ := result.Score("tone")
	fmt.Println("about billing:", billing.Noul > 0.85)
	fmt.Println("team:", team.Choice)
	fmt.Println("tone:", tones[tone.Level()])
	// Output:
	// about billing: true
	// team: billing
	// tone: annoyed
}

func ExampleClient_ListModels() {
	client, err := typesafe.New(typesafe.Config{APIKey: flag.Arg(0)})
	if err != nil {
		log.Fatal(err)
	}

	models, err := client.ListModels(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for _, m := range models {
		fmt.Println(m.Name)
	}
	// Output:
	// jev-latest
	// jev-preview
}
