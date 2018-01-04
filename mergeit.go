package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/github"
	"github.com/k0kubun/pp"
	"golang.org/x/oauth2"
)

func main() {
	fmt.Println("-- mergeit! --")

	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GH_ACCESS_TOKEN")},
	)
	httpClient := oauth2.NewClient(ctx, tokenSource)
	client := github.NewClient(httpClient)
	listOpts := &github.ListOptions{PerPage: 100}

	pullRequest, _, err := client.PullRequests.Get(ctx, "pmdarrow", "test", 1)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		return
	}

	fmt.Printf("Mergable: %v\n", pullRequest.GetMergeable())
	fmt.Printf("MergableState: %v\n", pullRequest.GetMergeableState())

	statuses, _, err := client.Repositories.GetCombinedStatus(
		ctx, "pmdarrow", "test", *pullRequest.Head.SHA, listOpts)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		return
	}

	pp.Print(statuses)
}
