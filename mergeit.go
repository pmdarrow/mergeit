package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/k0kubun/pp"
	"github.com/shomali11/slacker"
	"golang.org/x/oauth2"
)

const retryTimeout = time.Second * 30

var (
	Info  *log.Logger
	Error *log.Logger

	// TODO: Is storing these outside main() okay?
	ctx      context.Context
	client   *github.Client
	listOpts github.ListOptions
	bot      *slacker.Slacker
)

func init() {
	Info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	Error = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime)
}

func main() {
	Info.Println("-- mergeit bot --")

	ctx = context.Background()
	token := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GH_ACCESS_TOKEN")},
	)
	oauth := oauth2.NewClient(ctx, token)
	client = github.NewClient(oauth)
	listOpts = github.ListOptions{PerPage: 100}
	bot = slacker.NewClient(os.Getenv("SLACK_ACCESS_TOKEN"))

	bot.Init(func() {
		Info.Println("Connected to Slack, listening for commands.")
	})

	bot.Err(func(err string) {
		Error.Println("", err)
	})

	bot.Command(
		"merge <pr-url> <merge-method>",
		"Keep a pull request up-to-date and merge it when successfully built.",
		func(request *slacker.Request, response slacker.ResponseWriter) {
			response.Reply("Will do!")
			prURL := strings.Trim(request.StringParam("pr-url", ""), "<>")
			mergeMethod := strings.Trim(request.StringParam("merge-method", "squash"), "<>")
			err := mergeitURL(prURL, mergeMethod)
			if err != nil {
				response.Reply(fmt.Sprintf("Error merging %v: %v", prURL, err))
				return
			}
			response.Reply(fmt.Sprintf("%v successfully merged.", prURL))
		})

	bot.Default(func(request *slacker.Request, response slacker.ResponseWriter) {
		response.Reply("Say what? Try 'help'")
	})

	err := bot.Listen()
	if err != nil {
		log.Fatal(err)
	}
}

func mergeitURL(prURL string, mergeMethod string) error {
	parsed, err := url.Parse(prURL)
	if err != nil {
		Error.Println(err)
		return err
	}

	path := strings.Split(parsed.Path, "/")
	if len(path) != 5 {
		msg := "URL doesn't look right. Ensure it's a full GitHub PR URL without a trailing slash."
		Error.Println(msg)
		return errors.New(msg)
	}

	owner := path[1]
	repo := path[2]
	prNum, err := strconv.Atoi(path[4])
	if err != nil {
		Error.Println(err)
		return err
	}

	return mergeit(owner, repo, prNum, mergeMethod)
}

func mergeit(owner string, repo string, prNum int, mergeMethod string) error {
	Info.Printf("Fetching PR #%v from %v/%v...\n", prNum, owner, repo)

	pullRequest, _, err := client.PullRequests.Get(ctx, owner, repo, prNum)
	if err != nil {
		Error.Println(err)
		return err
	}

	statuses, _, err := client.Repositories.GetCombinedStatus(
		ctx, owner, repo, *pullRequest.Head.SHA, &listOpts)
	if err != nil {
		Error.Println(err)
		return err
	}

	if pullRequest.GetMerged() {
		// TODO: This pattern seems repetitive - should I just be returning
		// error and doing logging outside of this function? Doesn't seem like it
		// would be consistent with Info logging if I did that...
		msg := "PR already merged!"
		Error.Printf(msg)
		return errors.New(msg)
	}

	if pullRequest.GetMergeableState() == "unknown" {
		Info.Printf("Merge state unknown, retrying in %v.\n", retryTimeout)
		time.Sleep(retryTimeout)
		return mergeit(owner, repo, prNum, mergeMethod)
	}

	if pullRequest.GetMergeableState() == "behind" {
		Info.Println("PR is out-of-date; merging the latest changes from master.")
		request := &github.RepositoryMergeRequest{
			// TODO: can't use GetLabel here because it returns string... should I
			// be dereferencing like *pullRequest.Merged instead of GetMerged()
			// elsewhere?
			Base: pullRequest.Head.Ref,
			Head: pullRequest.Base.Ref,
		}
		_, _, err := client.Repositories.Merge(ctx, owner, repo, request)
		if err != nil {
			Error.Println(err)
			return err
		}
		Info.Printf("PR up-to-date, starting over in %v.\n", retryTimeout)
		time.Sleep(retryTimeout)
		return mergeit(owner, repo, prNum, mergeMethod)
	}

	if pullRequest.GetMergeableState() == "dirty" {
		msg := "PR has conflicts that must be manually resolved."
		Error.Println(msg)
		return errors.New(msg)
	}

	if statuses.GetState() == "failed" {
		msg := "Build failed and must be manually fixed."
		Error.Println(msg)
		return errors.New(msg)
	}

	if statuses.GetState() == "pending" {
		Info.Printf("Build in progress. Retrying in %v.\n", retryTimeout)
		time.Sleep(retryTimeout)
		return mergeit(owner, repo, prNum, mergeMethod)
	}

	if pullRequest.GetMergeableState() == "clean" && statuses.GetState() == "success" {
		Info.Printf("Ready to be merged! Merging with the \"%v\" method.\n", mergeMethod)
		// XXX: Provide proper squash message here
		message := ""

		opts := &github.PullRequestOptions{
			MergeMethod: mergeMethod,
		}
		_, _, err := client.PullRequests.Merge(ctx, owner, repo, prNum, message, opts)
		if err != nil {
			Error.Println(err)
			return err
		}

		Info.Println("PR successfully merged.")
		return nil
	}

	msg := "Reached end of mergeit() without doing anything. Something's wrong."
	Error.Println(msg)
	pp.Print(pullRequest)
	pp.Print(statuses)
	return errors.New(msg)
}
