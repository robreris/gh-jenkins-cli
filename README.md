# FortinetCloudCSE Go CLI Tool

This is a CLI tool to work with repos and Jenkins pipelines in the FortinetCloudCSE org. 

```
# Clone the repo
git clone <URL>

# Compile for OS (linux/mac/windows)
GOOS=linux GOARCH=amd64 go build -o gh-jenkins-cli 

# Populate setenv-template.sh and run it to set environment variables
source setenv-template.sh

# Create GitHub Repo based on UserRepo template
./gh-jenkins-cli create-repo --name test-repo

# Create a Jenkins pipeline for it
./gh-jenkins-cli create-job --name test-repo
```
