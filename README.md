## FortinetCloudCSE CI/CD Helper

This is a Go-based CLI tool to work with repos and Jenkins pipelines in the FortinetCloudCSE org. 

It includes functionalities for working with:

* FortinetCloudCSE GitHub Repos
* Jenkins Pipeline/Job Creation/Deletion

To **build** the tool, you'll need a recent [Golang](https://go.dev/) version installed. Golang installation information may be found [here](https://go.dev/doc/install).

*Clone the repository*
```
git clone <HTTPS/SSH URL found in the 'Code' dropdown above>

cd gh-jenkins-cli
```

*Download necessary go libraries:*
```
go mod download
```

*Build:*

**Note: Before building, you can confirm availability of the desired OS/Architecture via:**
```
go tool dist list
``` 

- **Linux/x86_64:**
```
GOOS=linux GOARCH=amd64 go build -o gh-jenkins-cli .
```
- **macOS/AMD64:**
```
GOOS=darwin GOARCH=amd64 go build -o gh-jenkins-cli .
```
- **Windows/x86_64:**
```
GOOS=windows GOARCH=amd64 go build -o gh-jenkins-cli.exe .

```

*Update executable permissions if needed:*
```
> chmod +x gh-jenkins-cli
```

## Using the Tool

In order to use the tool, you'll need a GitHub personal access token with all boxes under repo, admin:repo_hook, and delete_repo permissions checked.

You'll also need a Jenkins API token. Log in to Jenkins, click your username at the top right of the screen, click **Security**, then **Add new Token**, give it a name, and click **Generate**. Save the token to a safe place. Once you have these items, you can populate the provided script to set the environment variables needed for the tool to run.

```bash
mv setenv-template.sh setenv.sh

# Populate the script and run it.
source setenv.sh

# Confirm the environment variables have been set.
: "${GITHUB_TOKEN?} ${JENKINS_URL?} ${JENKINS_USER_ID?} ${JENKINS_API_TOKEN?}"

# Usage:
./gh-jenkins-cli [command] [flags]

# You can access a help menu by running:
./gh-jenkins-cli -h

```

### Available Commands

| Command         | Description                                                 |
|-----------------|-------------------------------------------------------------|
| create-repo     | Create a GitHub repo in the FortinetCloudCSE org.           |
| create-job      | Create a Jenkins job associated with a repo.                |
| create-project  | Create a GitHub repo and Jenkins job.                       |
| delete-project  | Delete a GitHub repo and its associated Jenkins job.        |
| add-collab      | Add collaborators to a GitHub repo.                         |
| delete-job      | Delete an existing Jenkins job.                             |
| delete-repo     | Delete an existing GitHub repo in the FortinetCloudCSE org. |

### Examples
```bash
# Create a new GitHub repo
./gh-jenkins-cli create-repo -n my-new-repo

# Add collaborators with push (default) permissions
./gh-jenkins-cli add-collab -c user1,user2,user3 -r my-new-repo

# Create an associated Jenkins job
./gh-jenkins-cli create-job -n my-new-repo

# Delete an entire project (Jenkins job + GitHub repo)
./gh-jenkins-cli delete-project -p my-new-repo
```
