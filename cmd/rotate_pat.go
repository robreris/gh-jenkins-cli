package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/robreris/gh-jenkins-cli/jenkins"
	"github.com/spf13/cobra"
)

var (
	credentialID string
	patSecret    string
	credDesc     string
	expiresDate  string
)

const patExpiryFile = ".config/gh-jenkins-cli/pat_expiry"

const bashrcMarker = "# BEGIN gh-jenkins-cli pat reminder"

const bashrcSnippet = `
# BEGIN gh-jenkins-cli pat reminder
if [ -f "$HOME/.config/gh-jenkins-cli/pat_expiry" ]; then
  _pat_expiry=$(cat "$HOME/.config/gh-jenkins-cli/pat_expiry")
  _pat_expiry_ts=$(date -d "$_pat_expiry" +%s 2>/dev/null)
  _today_ts=$(date +%s)
  _days_left=$(( (_pat_expiry_ts - _today_ts) / 86400 ))
  if [ "$_days_left" -le 0 ]; then
    echo "WARNING: GitHub PAT has expired ($_pat_expiry). Run: gh-jenkins-cli rotate-pat"
  elif [ "$_days_left" -le 14 ]; then
    echo "WARNING: GitHub PAT expires in $_days_left day(s) ($_pat_expiry). Run: gh-jenkins-cli rotate-pat"
  fi
  unset _pat_expiry _pat_expiry_ts _today_ts _days_left
fi
# END gh-jenkins-cli pat reminder`

var rotatePatCmd = &cobra.Command{
	Use:   "rotate-pat",
	Short: "Create or update a Jenkins Secret Text credential with a new GitHub PAT value",
	Long: `Upserts a Jenkins Secret Text credential identified by --credential-id.
If the credential already exists its secret value is updated.
If it does not exist a new credential is created.

The PAT value can be passed via --secret or read from the GITHUB_TOKEN env var.

Use --expires (YYYY-MM-DD) to save the PAT expiry date. A warning will appear
in new terminal windows when expiry is 14 days or less away.`,
	Run: func(cmd *cobra.Command, args []string) {
		if patSecret == "" {
			patSecret = os.Getenv("GITHUB_TOKEN")
		}
		if patSecret == "" {
			log.Fatal("PAT value required: provide --secret or set GITHUB_TOKEN env var")
		}

		if expiresDate != "" {
			if _, err := time.Parse("2006-01-02", expiresDate); err != nil {
				log.Fatal("--expires must be in YYYY-MM-DD format")
			}
		}

		client := jenkins.NewAPIClient()

		if err := client.UpsertSecretTextCredential(credentialID, patSecret, credDesc); err != nil {
			log.Fatal("Error upserting Jenkins credential: ", err)
		}

		fmt.Printf("Credential '%s' upserted in Jenkins.\n", credentialID)

		if err := client.UpdateGitHubServerCredential(credentialID); err != nil {
			log.Fatal("Error updating Jenkins GitHub server credential: ", err)
		}

		fmt.Printf("GitHub server configuration updated to use credential '%s'.\n", credentialID)

		if expiresDate != "" {
			if err := savePatExpiry(expiresDate); err != nil {
				log.Printf("Warning: could not save expiry date: %v", err)
			} else if err := ensureBashrcSnippet(); err != nil {
				log.Printf("Warning: could not update .bashrc: %v", err)
			} else {
				fmt.Printf("Expiry reminder set for %s.\n", expiresDate)
			}
		}
	},
}

func savePatExpiry(date string) error {
	dir := filepath.Join(os.Getenv("HOME"), ".config", "gh-jenkins-cli")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %v", err)
	}
	path := filepath.Join(os.Getenv("HOME"), patExpiryFile)
	return os.WriteFile(path, []byte(date), 0644)
}

func ensureBashrcSnippet() error {
	bashrcPath := filepath.Join(os.Getenv("HOME"), ".bashrc")
	data, err := os.ReadFile(bashrcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .bashrc: %v", err)
	}
	if strings.Contains(string(data), bashrcMarker) {
		return nil
	}
	f, err := os.OpenFile(bashrcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .bashrc: %v", err)
	}
	defer f.Close()
	_, err = f.WriteString(bashrcSnippet + "\n")
	return err
}

func init() {
	rootCmd.AddCommand(rotatePatCmd)
	rotatePatCmd.Flags().StringVarP(&credentialID, "credential-id", "i", "", "Jenkins credential ID to create or update")
	rotatePatCmd.Flags().StringVarP(&patSecret, "secret", "s", "", "New PAT value (defaults to GITHUB_TOKEN env var if omitted)")
	rotatePatCmd.Flags().StringVarP(&credDesc, "description", "d", "GitHub PAT", "Description for the Jenkins credential")
	rotatePatCmd.Flags().StringVarP(&expiresDate, "expires", "e", "", "PAT expiration date in YYYY-MM-DD format; sets a terminal reminder")
	rotatePatCmd.MarkFlagRequired("credential-id")
}
