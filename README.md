# github-issue-to-jira

This action helps to sync github issue to Jira automatically:

- An example of using a this action is when a label is applied to a GitHub issue, triggering the action to automatically synchronize the issue with Jira.

- A custom label can be utilized as a sync/skipping.

# V1

The V1 edition of the action offers:

- Sync Github issue to Jira
- A custom acceptance label can be utilized as a synchronization target.
- A custom synced label can be utilized as a synchronization　skipping.
- By applying a custom Jira issue type to a Jira ticket, it can be utilized.

The action will first check if the labeled issue exists, then fetch the attached labels to check if the issue is already accepted by the organizer or checked already synced to Jira.
If the already attached synced label, the action will skip the whole process. Otherwise, the action will create a new Jira issue based on the Github issue and send a request to Jira to create a new Jira issue by using Jira rest API.

**Note:** 
- The `github-issue-to-jira` action executes GitHub issues such as read/write. So your provided GitHub token should have read/write authorization.

# Usage

See [action.yaml](action.yaml)

## Basic

```yaml
name: Sync issues to jira
on:
  issues:
    types: [labeled]　# should be set
jobs:
  issue-sync:
    runs-on: ubuntu-latest
steps:
  - uses: actions/checkout@v3
  - name: Set up Go
    uses: actions/setup-go@v4
    with:
      go-version: '>=1.19.0' # The Go version should be grater than 1.19.0
  - name: Sync to Jira
    uses: atsushii/github-issue-to-jira@v1
    with:
      github-owner: github-owner
      github-repo: repository-name
      github-token: ${{ secrets.GITHUB_TOKEN }} # read/write authorized token
      jira-hostname: hostname # Your Sync dest Jira hostname
      jira-auth-token: ${{ secrets.JIRA_AUTH_TOKEN }} # Your Jira auth token
      jira-auth-email: ${{ secrets.JIRA_AUTH_EMAIL }} # email same as jira project creator
      jira-project-key: project-key # Your Sync dest Jira project name
```

## Using Custom Github issue label to sync/skip

The `accepted-label` flag defaults to `triage/accepted`. Use the default or set your custom label to an acceptance label if you prefer.

The `synced-label` flag defaults to `workflow/synced`. Use the default or set your custom label to an acceptance label if you prefer.

```yaml
name: Sync issues to jira
on:
  issues:
    types: [labeled]　# should be set
jobs:
  issue-sync:
    runs-on: ubuntu-latest
steps:
  - uses: actions/checkout@v3
  - name: Set up Go
    uses: actions/setup-go@v4
    with:
      go-version: '>=1.19.0' # The Go version should be grater than 1.19.0
  - name: Sync to Jira
    uses: atsushii/github-issue-to-jira@v1
    with:
      github-owner: github-owner
      github-repo: repository-name
      github-token: ${{ secrets.GITHUB_TOKEN }} # read/write authorized token
      jira-hostname: hostname # Your Sync dest Jira hostname
      jira-auth-token: ${{ secrets.JIRA_AUTH_TOKEN }} # Your Jira auth token
      jira-auth-email: ${{ secrets.JIRA_AUTH_EMAIL }} # email same as jira project creator
      jira-project-key: project-key # Your Sync dest Jira project name
      synced-label: custom-synced-label
      accepted-label: custom-accepted-label
```

## Using Custom Jira Issue Type

The `jira-issue-type` flag defaults to `Bug`. Use the default or set your custom issue type if you prefer.

**Note:** Set custom issue type, you need to create exact same issue type on Jira project first otherwise, Jira API will return 400.

```yaml
name: Sync issues to jira
on:
  issues:
    types: [labeled]　# should be set
jobs:
  issue-sync:
    runs-on: ubuntu-latest
steps:
  - uses: actions/checkout@v3
  - name: Set up Go
    uses: actions/setup-go@v4
    with:
      go-version: '>=1.19.0' # The Go version should be grater than 1.19.0
  - name: Sync to Jira
    uses: atsushii/github-issue-to-jira@v1
    with:
      github-owner: github-owner
      github-repo: repository-name
      github-token: ${{ secrets.GITHUB_TOKEN }} # read/write authorized token
      jira-hostname: hostname # Your Sync dest Jira hostname
      jira-auth-token: ${{ secrets.JIRA_AUTH_TOKEN }} # Your Jira auth token
      jira-auth-email: ${{ secrets.JIRA_AUTH_EMAIL }} # email same as jira project creator
      jira-project-key: project-key # Your Sync dest Jira project name
      jira-issue-type: custom-jira-issue-type
```

## Set Specific Github Issue number

The `github-issue-number` flag defaults to `${{ github.event.issue.number }}`. Which can get labeled issues internally, However, allow you to set specific issue numbers if you prefer.

```yaml
name: Sync issues to jira
on:
  issues:
    types: [labeled]　# should be set
jobs:
  issue-sync:
    runs-on: ubuntu-latest
steps:
  - uses: actions/checkout@v3
  - name: Set up Go
    uses: actions/setup-go@v4
    with:
      go-version: '>=1.19.0' # The Go version should be grater than 1.19.0
  - name: Sync to Jira
    uses: atsushii/github-issue-to-jira@v1
    with:
      github-owner: github-owner
      github-repo: repository-name
      github-token: ${{ secrets.GITHUB_TOKEN }} # read/write authorized token
      jira-hostname: hostname # Your Sync dest Jira hostname
      jira-auth-token: ${{ secrets.JIRA_AUTH_TOKEN }} # Your Jira auth token
      jira-auth-email: ${{ secrets.JIRA_AUTH_EMAIL }} # email same as jira project creator
      jira-project-key: project-key # Your Sync dest Jira project name
      github-issue-number: 1
```

# License

The scripts and documentation in this project are released under the [MIT License](LICENSE)
