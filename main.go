package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
)

type IssueName struct {
	Name string `json:"name"`
}

type IssueValue struct {
	Value string `json:"value"`
}

type IssueKey struct {
	Key string `json:"key"`
}

type IssueFields struct {
	Project     IssueKey   `json:"project"`
	Summary     string     `json:"summary"`
	Description string     `json:"description"`
	IssueUrl    IssueValue `json:"customfield_10016"`
	IssueType   IssueName  `json:"issuetype"`
}

type NewJiraIssue struct {
	Fields IssueFields `json:"fields"`
}

type IssueCreationResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

type IssueCreationResult struct {
	Success []NewJiraIssue `json:"success"`
	Failed  []NewJiraIssue `json:"failed"`
}

func main() {
	ctx := context.Background()

	githubRepositoryOwner := os.Getenv("GITHUB_OWNER")
	githubRepositoryName := os.Getenv("GITHUB_REPO")
	githubAccessToken := os.Getenv("GITHUB_TOKEN")
	githubIssueNumber := os.Getenv("GITHUB_ISSUE_NUMBER")
	jiraProjectKey := os.Getenv("JIRA_PROJECT_KEY")
	jiraHostname := os.Getenv("JIRA_HOSTNAME")
	jiraAuthToken := os.Getenv("JIRA_AUTH_TOKEN")
	jiraIssueType := os.Getenv("JIRA_ISSUE_TYPE")
	syncedLabel := os.Getenv("SYNCED_LABEL")
	acceptedLabel := os.Getenv("ACCEPTED_LABEL")
	fmt.Printf("********** %s", githubRepositoryOwner)

	if githubRepositoryOwner == "" {
		log.Fatal("GITHUB_OWNER not set")
	}

	if githubRepositoryName == "" {
		log.Fatal("GITHUB_REPO not set")
	}

	if githubAccessToken == "" {
		log.Fatal("GITHUB_TOKEN not set")
	}

	if jiraProjectKey == "" {
		log.Fatal("JIRA_PROJECT_KEY not set")
	}

	if jiraHostname == "" {
		log.Fatal("JIRA_HOSTNAME not set")
	}

	if jiraAuthToken == "" {
		log.Fatal("JIRA_AUTH_TOKEN not set")
	}

	issueNumber, err := strconv.Atoi(githubIssueNumber)
	if err != nil {
		log.Fatalf("error parse provided perPage: %s, err: %s", githubIssueNumber, err)
	}

	client := newGithubClient(ctx, githubAccessToken)

	issue, _, err := client.Issues.Get(ctx, githubRepositoryOwner, githubRepositoryName, issueNumber)
	if err != nil {
		log.Fatalf("error retrieving issue %s/%s#%d: %s", githubRepositoryOwner, githubRepositoryName, issueNumber, err)
	}

	if hasLabel(issue, syncedLabel) {
		log.Printf("issue is already marked as synced (%s), skipping", syncedLabel)
		os.Exit(0)
	}

	if !hasLabel(issue, acceptedLabel) {
		log.Printf("issue is not marked as ready for syncing using %s, skipping", acceptedLabel)
		os.Exit(0)
	}

	newIssue := NewJiraIssue{Fields: IssueFields{
		Project:     IssueKey{Key: jiraProjectKey},
		Summary:     *issue.Title,
		Description: jirafyBodyMarkdown(issue),
		IssueUrl:    IssueValue{Value: *issue.URL},
		IssueType:   IssueName{Name: jiraIssueType},
	}}

	err = createJiraIssue(newIssue, jiraHostname, jiraAuthToken)

	if err != nil {
		log.Fatalf("error create new issue to jira %s/%s#%d: %s", githubRepositoryOwner, githubRepositoryName, issueNumber, err)
	}

	_, _, err = client.Issues.AddLabelsToIssue(ctx, githubRepositoryOwner, githubRepositoryName, issueNumber, []string{syncedLabel})
	if err != nil {
		log.Printf("error adding synced label for issue %s/%s#%d: %s", githubRepositoryOwner, githubRepositoryName, issueNumber, err)
	}

	os.Exit(0)
}

func hasLabel(issue *github.Issue, label string) bool {
	for _, l := range issue.Labels {
		if *l.Name == label {
			return true
		}
	}

	return false
}

func jirafyBodyMarkdown(issue *github.Issue) string {
	output := "GitHub issue: " + *issue.HTMLURL + "\n\n---\n\n"

	output += *issue.Body
	output = strings.ReplaceAll(output, "- [X] ", "✅ ")
	output = strings.ReplaceAll(output, "###", "h3.")
	output = strings.ReplaceAll(output, "```hcl", "{code}")
	output = strings.ReplaceAll(output, "```", "{code}")

	return output
}

func newGithubClient(ctx context.Context, accessToken string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}

func createJiraIssue(issue NewJiraIssue, jiraHostname, jiraAuthToken string) error {
	res, err := json.Marshal(issue)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s/rest/api/latest/issue/", jiraHostname)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(res))
	if err != nil {
		return err
	}

	req.Header.Set("authorization", "Basic "+jiraAuthToken)
	req.Header.Set("content-type", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var createdIssue IssueCreationResponse
	json.Unmarshal([]byte(body), &createdIssue)

	if resp.StatusCode != http.StatusCreated {
		return err
	}

	fmt.Printf("successfully created internal JIRA issue: %s", createdIssue.Key)

	return nil
}
