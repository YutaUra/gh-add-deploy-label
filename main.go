package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

type GitHubLabel struct {
	Name string `json:"name"`
}

type GetPullRequestResponse struct {
	Title   string        `json:"title"`
	HtmlUrl string        `json:"html_url"`
	Labels  []GitHubLabel `json:"labels"`
}

func getPullRequest(client *api.RESTClient, repo repository.Repository, prNumber string) (*GetPullRequestResponse, error) {
	var pull *GetPullRequestResponse
	err := client.Get(fmt.Sprintf("repos/%s/%s/pulls/%s", repo.Owner, repo.Name, prNumber), &pull)
	if err != nil {
		return nil, fmt.Errorf("could not get pull request: %w", err)
	}
	return pull, nil
}

func removeDeployLabel(client *api.RESTClient, repo repository.Repository, prNumber string) error {
	var resp interface{}
	err := client.Delete(fmt.Sprintf("repos/%s/%s/issues/%s/labels/deploy", repo.Owner, repo.Name, prNumber), resp)
	if err != nil {
		return fmt.Errorf("could not delete deploy label: %w", err)
	}
	return nil
}

func addDeployLabel(client *api.RESTClient, repo repository.Repository, prNumber string) error {
	json := "{\"labels\":[\"deploy\"]}"
	body := strings.NewReader(json)
	var resp interface{}
	err := client.Post(fmt.Sprintf("repos/%s/%s/issues/%s/labels", repo.Owner, repo.Name, prNumber), body, resp)
	if err != nil {
		return fmt.Errorf("could not add deploy label: %w", err)
	}
	return nil
}

func main() {
	err := cli()
	if err != nil {
		log.Fatal(err)
	}
}

func cli() error {
	repoOverride := flag.String(
		"repo", "", "Specify a repository. If omitted, uses current repository")
	flag.Parse()

	var repo repository.Repository
	var err error

	if *repoOverride == "" {
		repo, err = repository.Current()
	} else {
		repo, err = repository.Parse(*repoOverride)
	}
	if err != nil {
		return fmt.Errorf("could not determine what repo to use: %w", err)

	}

	if len(flag.Args()) < 1 {
		return errors.New("pr number is missing")
	}
	prNumber := strings.Join(flag.Args(), " ")

	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}

	pull, err := getPullRequest(client, repo, prNumber)
	if err != nil {
		return fmt.Errorf("could not get pull request: %w", err)
	}

	fmt.Printf("Adding deploy label to PR [%s](%s)\n", pull.Title, pull.HtmlUrl)

	// if already has the deploy label, remove it
	for _, label := range pull.Labels {
		if label.Name == "deploy" {
			err = removeDeployLabel(client, repo, prNumber)
			if err != nil {
				return fmt.Errorf("could not remove deploy label: %w", err)
			}
			break
		}
	}

	// add the deploy label
	err = addDeployLabel(client, repo, prNumber)
	if err != nil {
		return fmt.Errorf("could not add deploy label: %w", err)
	}

	fmt.Println("Done!Enjoy your environment!")
	return nil
}
