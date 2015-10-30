package main

import (
	"fmt"
	"github.com/google/go-github/github"
	git "github.com/libgit2/git2go"
	"golang.org/x/oauth2"
	"os"
	"path/filepath"
	"time"
)

const targetDir string = "starred"

var githubApiKey string

func updateRepositories(last time.Time) (*time.Time, error) {

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubApiKey},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)
	now := time.Now()

	options := &github.ActivityListStarredOptions{Sort: "updated"}
	for page := 1; ; page++ {
		options.Page = page

		starred, res, err := client.Activity.ListStarred("", options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not acquire starred repositories: %s", err)
			return nil, err
		}

		for _, repo := range starred {
			repoPath := filepath.Join(targetDir, *repo.Repository.Name)

			f, err := os.Open(repoPath)
			doesNotExist := false
			if err != nil && os.IsNotExist(err) {
				doesNotExist = true
			} else {
				f.Close()
			}

			if doesNotExist {
				_, err := git.Clone(*repo.Repository.CloneURL, repoPath, &git.CloneOptions{})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Could not clone repository %s: %s", repo, err)
					continue
				}
			} else {
				updateNeeded := repo.Repository.PushedAt.Time.After(last)
				if updateNeeded {
					r, err := git.OpenRepository(repoPath)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Could not open repository %s: %s", repoPath, err)
						continue
					}

					remote, err := r.Remotes.Lookup("origin")
					if err != nil {
						fmt.Fprintf(os.Stderr, "Could not get remote repository: %s", err)
						continue
					}

					err = remote.Fetch([]string{""}, nil, "")
					if err != nil {
						fmt.Fprintf(os.Stderr, "Could not fetch new objets: %s", err)
						continue
					}
				}
			}

			if page >= res.LastPage {
				break
			}
		}
	}
	return &now, nil
}

func main() {
	var ok bool

	githubApiKey, ok = os.LookupEnv("GITHUB_API_KEY")
	if !ok {
		fmt.Fprintf(os.Stderr, "No GITHUB_API_KEY environment variable")
		return
	}

	prev := time.Now()
	for {
		newUpdate, err := updateRepositories(prev)
		if err == nil {
			prev = *newUpdate
		}
		time.Sleep(20 * time.Minute)
	}
}
