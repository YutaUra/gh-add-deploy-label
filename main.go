package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

type GitHubLabel struct {
	Name string `json:"name"`
}

type GitHubPullRequest struct {
	Number  int           `json:"number"`
	Title   string        `json:"title"`
	HtmlUrl string        `json:"html_url"`
	Labels  []GitHubLabel `json:"labels"`
}

func getPullRequest(client *api.RESTClient, repo repository.Repository, prNumber string) (*GitHubPullRequest, error) {
	var pull *GitHubPullRequest
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

func getCurrentBranch() (string, error) {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return "", fmt.Errorf("could not get current branch: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", errors.New("could not get current branch. get empty string")
	}
	return branch, nil
}

func currentPullNumber(client *api.RESTClient, repo repository.Repository) (string, error) {
	branch, err := getCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("could not get current branch: %w", err)
	}

	var pulls *[]GitHubPullRequest
	err = client.Get(fmt.Sprintf("repos/%s/%s/pulls?head=%s", repo.Owner, repo.Name, branch), &pulls)
	if err != nil {
		return "", fmt.Errorf("could not get pull requests: %w", err)
	}
	for _, pull := range *pulls {
		return fmt.Sprintf("%d", pull.Number), nil
	}
	return "", errors.New("could not get pull number")
}

func main() {
	err := cli()
	if err != nil {
		log.Fatal(err)
	}
}

func cli() error {
	var repo repository.Repository

	repo, err := repository.Current()
	if err != nil {
		return fmt.Errorf("could not determine what repo to use: %w", err)
	}

	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}

	var prNumber string

	if len(os.Args) <= 1 {
		fmt.Println("No pull number specified. Using current branch.")
		prNumber, err = currentPullNumber(client, repo)
		if err != nil {
			return fmt.Errorf("could not get current pull number: %w", err)
		}
	} else {
		fmt.Printf("Using pull number from arguments '%s'.\n", os.Args[1])
		prNumber = os.Args[1]
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
