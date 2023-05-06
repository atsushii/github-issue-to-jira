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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
)

var (
	since = time.Now().Add(3 * time.Minute)
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
	Project     IssueKey     `json:"project"`
	Summary     string       `json:"summary"`
	Description string       `json:"description"`
	IssueUrl    IssueValue `json:"customfield_10016"`
	IssueType   IssueName    `json:"issuetype"`
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
	Failed []NewJiraIssue `json:"failed"`
}

func main() {
	ctx := context.Background()

	githubRepositoryOwner := os.Getenv("GITHUB_OWNER")
	githubRepositoryName := os.Getenv("GITHUB_REPO")
	githubAccessToken := os.Getenv("GITHUB_TOKEN")
	githubIssueNumber := os.Getenv("INPUT_GITHUB_ISSUE_NUMBER")
	jiraProjectKey := os.Getenv("JIRA_PROJECT_KEY")
	jiraHostname := os.Getenv("JIRA_HOSTNAME")
	jiraAuthToken := os.Getenv("JIRA_AUTH_TOKEN")
	accessClientID := os.Getenv("CF_ACCESS_CLIENT_ID")
	accessClientSecret := os.Getenv("CF_ACCESS_CLIENT_SECRET")
	jiraIssueType := os.Getenv("JIRA_ISSUE_TYPE")
	syncedLabel := os.Getenv("SYNCED_LABEL")
	acceptedLabel := os.Getenv("ACCEPTED_LABEL")
	inputSince := os.Getenv("SINCE")
	inputPerPage := os.Getenv("PER_PAGE")
	fmt.Printf("********** %s", githubIssueNumber)

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

	if accessClientID == "" {
		log.Fatal("CF_ACCESS_CLIENT_ID not set")
	}

	if accessClientSecret == "" {
		log.Fatal("CF_ACCESS_CLIENT_SECRET not set")
	}

	if inputSince != "" {
		sinceTime, err := time.Parse("2006-01-02T15:04:05Z", inputSince)
		if err != nil {
			log.Fatalf("error parse provided since: %s, err: %s",inputSince, err)
		}
		since = sinceTime
	}

	perPage, err := strconv.Atoi(inputPerPage)
		if err != nil {
			log.Fatalf("error parse provided perPage: %s, err: %s",inputPerPage, err)
		}

	client := newGithubClient(ctx, githubAccessToken)

	// get latest labeled issue
	issues, _, err := client.Issues.ListByRepo(ctx, githubRepositoryOwner, githubRepositoryName, &github.IssueListByRepoOptions{
		Sort:   "updated",
		Direction: "desc",
		Labels: []string{acceptedLabel},
		Since: since,
		ListOptions: github.ListOptions{
			PerPage: perPage,
		},
	})

	if err != nil {
		log.Fatalf("error retrieving latest labeled issues %s/%s: %s", githubRepositoryOwner, githubRepositoryName, err)
	}

	if len(issues) == 0 {
		fmt.Println("no accepted issue detected; exiting without creating a new JIRA issue")
		os.Exit(0)
	}

	newIssues := newJiraIssues(ctx, client, issues, jiraProjectKey, jiraIssueType, acceptedLabel, syncedLabel)

	if len(newIssues) == 0 {
		fmt.Println("no unsynced issue is existing")
		os.Exit(0)
	}

	result := createJiraIssues(newIssues, jiraHostname, jiraAuthToken, accessClientID, accessClientSecret)

	for _, issue := range result.Failed {
		log.Printf("error create issue. github issue url: %s, err: %s", issue.Fields.IssueUrl.Value, err)
	}

	for _, issue := range result.Success {
		issueNumber, err := getIssueNumber(issue.Fields.IssueUrl.Value)
		if err != nil {
			log.Printf("error adding synced label for issue %s/%s#%d: %s", githubRepositoryOwner, githubRepositoryName, issueNumber, err)
			continue
		}	

		_, _, err = client.Issues.AddLabelsToIssue(ctx, githubRepositoryOwner, githubRepositoryName, issueNumber, []string{syncedLabel})
		if err != nil {
			log.Printf("error adding synced label for issue %s/%s#%d: %s", githubRepositoryOwner, githubRepositoryName, issueNumber, err)
		}	
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

func newGithubClient(ctx context.Context, accessToken string)*github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}

func newJiraIssues(ctx context.Context, client *github.Client, githubIssues []*github.Issue, jiraProjectKey, jiraIssueType, acceptedLabel, syncedLabel string)[]NewJiraIssue {
	newIssues := make([]NewJiraIssue, 0, len(githubIssues))
	for _, issue := range githubIssues {	
		if hasLabel(issue, syncedLabel) {
			log.Printf("issue is already marked as synced (%s), skipping", syncedLabel)
			continue
		}
	
		if !hasLabel(issue, acceptedLabel) {
			log.Printf("issue is not marked as ready for syncing using %s, skipping", acceptedLabel)
			continue
		}

		newIssues = append(newIssues, NewJiraIssue{Fields: IssueFields{
			Project:     IssueKey{Key: jiraProjectKey},
			Summary:     *issue.Title,
			Description: jirafyBodyMarkdown(issue),
			IssueUrl: IssueValue{Value: *issue.URL},
			IssueType:   IssueName{Name: jiraIssueType},
		}})
	}
	return newIssues
}

func createJiraIssues(issues []NewJiraIssue, jiraHostname, jiraAuthToken, accessClientID, accessClientSecret string) IssueCreationResult {
	result := IssueCreationResult{}
	for _, issue := range issues {
		res, err := json.Marshal(issue)
		if err != nil {
			result.Failed = append(result.Failed, issue)
			continue
		}
	
		url := fmt.Sprintf("https://%s/rest/api/latest/issue/", jiraHostname)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(res))
		if err != nil {
			result.Failed = append(result.Failed, issue)
			continue
		}
	
		req.Header.Set("authorization", "Basic "+jiraAuthToken)
		req.Header.Set("cf-access-client-id", accessClientID)
		req.Header.Set("cf-access-client-secret", accessClientSecret)
		req.Header.Set("content-type", "application/json")
	
		httpClient := &http.Client{}
		resp, err := httpClient.Do(req)
		if err != nil {
			result.Failed = append(result.Failed, issue)
			continue
		}
		defer resp.Body.Close()
	
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			result.Failed = append(result.Failed, issue)
			continue
		}
	
		var createdIssue IssueCreationResponse
		json.Unmarshal([]byte(body), &createdIssue)
	
		if resp.StatusCode != http.StatusCreated {
			result.Failed = append(result.Failed, issue)
			continue
		}
		fmt.Printf("successfully created internal JIRA issue: %s", createdIssue.Key)
		result.Success = append(result.Success, issue)
	}
	return result
}

func getIssueNumber(issueUrl string) (int, error) {
	re := regexp.MustCompile(`/(\d+)$`)
	issueNumberStr := re.FindStringSubmatch(issueUrl)[1]

	issueNumber, err := strconv.Atoi(issueNumberStr)
	if err != nil {
		return 0, err
	}

	return issueNumber, nil
}
