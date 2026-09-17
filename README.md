# typesafe-go

A Go client for the [TypeSafe API](https://docs.typesafe.ai).

```sh
go get github.com/hhhapz/typesafe-go
```

```go
client, err := typesafe.New(typesafe.Config{
	APIKey: "...",
})
if err != nil {
	return err
}

result, err := client.SystemOne(ctx, typesafe.SystemOneRequest{
	State: map[string]any{
		"reference":      "Lasting for a very short time.",
		"learner_answer": "Something that doesn't last long.",
	},
	Questions: typesafe.Questions{
		"grade": typesafe.Score{
			Instructions: "How well does the learner answer match the reference?",
			Criteria:     []any{"Incorrect.", "Partially correct.", "Correct."},
		},
	},
})
if err != nil {
	return err
}

grade, _ := result.Score("grade")
```

Run the examples: `go test -v -run Example -args YOUR_API_KEY`.

Licensed under [MIT](LICENSE).
