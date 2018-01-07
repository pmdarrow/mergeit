package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// TODO: Is storing these outside main() okay?
var ctx context.Context
var client github.Client
var listOpts github.ListOptions

func init() {
	fmt.Println("-- mergeit! --")

	ctx = context.Background()
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GH_ACCESS_TOKEN")},
	)
	httpClient := oauth2.NewClient(ctx, tokenSource)
	client = *github.NewClient(httpClient)
	listOpts = github.ListOptions{PerPage: 100}
}

func main() {
	owner := "pmdarrow"
	repo := "test"
	pr := 1
	mergeit(owner, repo, pr)
}

func mergeit(owner string, repo string, pr int) {
	fmt.Printf("Fetching PR #%v from %v/%v...\n", pr, owner, repo)

	pullRequest, _, err := client.PullRequests.Get(ctx, owner, repo, pr)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if pullRequest.GetMerged() {
		fmt.Println("Error: PR already merged!")
		return
	}

	if pullRequest.GetMergeableState() == "dirty" {
		fmt.Println("Error: Branch has conflicts that must be manually resolved.")
		return
	}

	if pullRequest.GetMergeableState() == "behind" {
		fmt.Println("This branch is out-of-date with the base branch.")
		fmt.Println("Merging the latest changes from master into this branch...")

		request := &github.RepositoryMergeRequest{
			// TODO: can't use GetLabel here because it returns string... should I
			// be dereferencing like *pullRequest.Merged instead of GetMerged()
			// elsewhere?
			Base: pullRequest.Head.Ref,
			Head: pullRequest.Base.Ref,
		}
		_, _, err := client.Repositories.Merge(ctx, owner, repo, request)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println("Branch up-to-date, starting over...")
		mergeit(owner, repo, pr)
		return
	}

	if pullRequest.GetMergeableState() == "blocked" {
		statuses, _, err := client.Repositories.GetCombinedStatus(
			ctx, owner, repo, *pullRequest.Head.SHA, &listOpts)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if statuses.GetState() == "failed" {
			fmt.Println("Error: Build failed and must be manually fixed.")
			return
		}

		if statuses.GetState() == "pending" {
			fmt.Println("Waiting for build to complete...")
			// TODO: Add retrying
			return
		}
	}
}
