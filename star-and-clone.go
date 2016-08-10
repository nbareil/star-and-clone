package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
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
			glog.Errorf("Could not acquire starred repositories: %s", err)
			return nil, err
		}

		for _, repo := range starred {
			repoPath := filepath.Join(targetDir, *repo.Repository.Name)
			glog.V(2).Infof("Checking %s", *repo.Repository.Name)

			f, err := os.Open(repoPath)
			doesNotExist := false
			if err != nil && os.IsNotExist(err) {
				doesNotExist = true
			} else {
				f.Close()
			}

			if doesNotExist {
				glog.Infof("%s does not exist, cloning...", repoPath)
				_, err := git.Clone(*repo.Repository.CloneURL, repoPath, &git.CloneOptions{})
				if err != nil {
					glog.Errorf("Could not clone repository %s: %s", repo, err)
					continue
				}
			} else {
				updateNeeded := repo.Repository.PushedAt.Time.After(last)
				if updateNeeded {
					glog.Infof("%s needs updates", repoPath)
					r, err := git.OpenRepository(repoPath)
					if err != nil {
						glog.Errorf("Could not open repository %s: %s", repoPath, err)
						continue
					}

					remote, err := r.Remotes.Lookup("origin")
					if err != nil {
						glog.Errorf("Could not get remote repository: %s", err)
						continue
					}

					err = remote.Fetch([]string{""}, nil, "")
					if err != nil {
						glog.Errorf("Could not fetch new objets: %s", err)
						continue
					}
				} else {
					glog.V(2).Infof("%s is up-to-date", repoPath)
				}
			}
		}

		if page >= res.LastPage {
			glog.V(2).Infoln("Last page")
			break
		}
	}
	return &now, nil
}

func main() {
	var ok bool

	flag.Set("logtostderr", "true")
	flag.Parse()

	githubApiKey, ok = os.LookupEnv("GITHUB_API_KEY")
	if !ok {
		glog.Error("No GITHUB_API_KEY environment variable")
		return
	}

	prev := time.Now()
	for {
		glog.V(2).Infoln("Updating repositories")
		newUpdate, err := updateRepositories(prev)
		if err == nil {
			prev = *newUpdate
		}
		glog.V(2).Infoln("ZzzzzZzzzz")
		time.Sleep(20 * time.Minute)
	}
}
