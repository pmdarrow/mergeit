package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const retryTimeout = time.Second * 10

var (
	info  *log.Logger
	error *log.Logger

	// TODO: Is storing these outside main() okay?
	ctx      context.Context
	client   github.Client
	listOpts github.ListOptions
)

func init() {
	info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	error = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime)

	info.Println("-- mergeit! --")

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
	prNum := 9
	mergeMethod := "squash"
	mergeit(owner, repo, prNum, mergeMethod)
}

func mergeit(owner string, repo string, prNum int, mergeMethod string) {
	info.Printf("Fetching PR #%v from %v/%v...\n", prNum, owner, repo)

	pullRequest, _, err := client.PullRequests.Get(ctx, owner, repo, prNum)
	if err != nil {
		info.Println("Error:", err)
		return
	}

	if pullRequest.GetMerged() {
		info.Printf("Error: PR already merged!")
		return
	}

	if pullRequest.GetMergeableState() == "unknown" {
		info.Printf("Merge state unknown, retrying in %v.\n", retryTimeout)
		time.Sleep(retryTimeout)
		mergeit(owner, repo, prNum, mergeMethod)
		return
	}

	if pullRequest.GetMergeableState() == "dirty" {
		error.Println("Error: PR has conflicts that must be manually resolved.")
		return
	}

	if pullRequest.GetMergeableState() == "behind" {
		info.Println("PR is out-of-date; merging the latest changes from master.")

		request := &github.RepositoryMergeRequest{
			// TODO: can't use GetLabel here because it returns string... should I
			// be dereferencing like *pullRequest.Merged instead of GetMerged()
			// elsewhere?
			Base: pullRequest.Head.Ref,
			Head: pullRequest.Base.Ref,
		}
		_, _, err := client.Repositories.Merge(ctx, owner, repo, request)
		if err != nil {
			error.Println("Error:", err)
			return
		}
		info.Printf("PR up-to-date, starting over in %v.\n", retryTimeout)
		time.Sleep(retryTimeout)
		mergeit(owner, repo, prNum, mergeMethod)
		return
	}

	if pullRequest.GetMergeableState() == "blocked" {
		statuses, _, err := client.Repositories.GetCombinedStatus(
			ctx, owner, repo, *pullRequest.Head.SHA, &listOpts)
		if err != nil {
			error.Println("Error:", err)
			return
		}

		if statuses.GetState() == "success" {
			error.Println("Error: PR up-to-date and build passed, but still can't be merged.")
		}

		if statuses.GetState() == "failed" {
			error.Println("Error: Build failed and must be manually fixed.")
			return
		}

		if statuses.GetState() == "pending" {
			info.Printf("Build in progress. Retrying in %v.\n", retryTimeout)
			time.Sleep(retryTimeout)
			mergeit(owner, repo, prNum, mergeMethod)
			return
		}
	}

	if pullRequest.GetMergeableState() == "clean" {
		info.Printf("Ready to be merged! Merging with the \"%v\" method.\n", mergeMethod)
		// TODO: Provide proper squash message here
		message := "Merged by mergeit"

		opts := &github.PullRequestOptions{
			MergeMethod: mergeMethod,
		}
		_, _, err := client.PullRequests.Merge(ctx, owner, repo, prNum, message, opts)
		if err != nil {
			error.Println("Error:", err)
			return
		}

		info.Println("PR successfully merged.")
	}
}
