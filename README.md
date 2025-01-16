# FortinetCloudCSE Go CLI Tool

This is a CLI tool to work with repos and Jenkins pipelines in the FortinetCloudCSE org. 

```bash
# Clone the repo
git clone https://github.com/robreris/gh-jenkins-cli.git

# Compile for OS (linux/mac/windows); run 'go tool dist list' to see available GOOS/GOARCH pairs
GOOS=linux GOARCH=amd64 go build -o gh-jenkins-cli

# Populate setenv-template.sh and run it to set environment variables
source setenv-template.sh

# Create a Jenkins pipeline for the new repo
./gh-jenkins-cli create-job --name test-repo

# Create new GitHub repo based on UserRepo template
./gh-jenkins-cli create-repo --name test-repo


```
