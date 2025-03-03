package git

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	gitm "github.com/aymanbagabas/git-module"

	"github.com/dlvhdr/gh-dash/v4/utils"
)

// Extends git.Repository
type Repo struct {
	gitm.Repository
	Origin         string
	Remotes        []string
	Branches       []Branch
	HeadBranchName string
	Status         gitm.NameStatus
}

type Branch struct {
	Name          string
	LastUpdatedAt *time.Time
	CreatedAt     *time.Time
	LastCommitMsg *string
	CommitsAhead  int
	CommitsBehind int
	IsCheckedOut  bool
	Remotes       []string
}

func GetOriginUrl(dir string) (string, error) {
	repo, err := gitm.Open(dir)
	if err != nil {
		return "", err
	}
	remotes, err := repo.Remotes()
	if err != nil {
		return "", err
	}

	for _, remote := range remotes {
		if remote != "origin" {
			continue
		}

		urls, err := gitm.RemoteGetURL(dir, remote)
		if err != nil || len(urls) == 0 {
			return "", err
		}
		return urls[0], nil
	}

	return "", errors.New("no origin remote found")
}

func GetRepo(dir string) (*Repo, error) {
	repo, err := gitm.Open(dir)
	if err != nil {
		return nil, err
	}

	bNames, err := repo.Branches()
	if err != nil {
		return nil, err
	}

	headRef, err := repo.RevParse("HEAD", gitm.RevParseOptions{
		CommandOptions: gitm.CommandOptions{Args: []string{"--abbrev-ref"}},
	})
	if err != nil {
		return nil, err
	}
	status, err := getUnstagedStatus(repo)
	if err != nil {
		return nil, err
	}

	branches := make([]Branch, len(bNames))
	for i, b := range bNames {
		var updatedAt *time.Time
		var lastCommitMsg *string
		isHead := b == headRef
		commits, err := gitm.Log(dir, b, gitm.LogOptions{MaxCount: 1})
		if err == nil && len(commits) > 0 {
			updatedAt = &commits[0].Committer.When
			lastCommitMsg = utils.StringPtr(commits[0].Summary())
		}
		commitsAhead, err := repo.RevListCount([]string{fmt.Sprintf("origin/%s..%s", b, b)})
		if err != nil {
			commitsAhead = 0
		}
		commitsBehind, err := repo.RevListCount([]string{fmt.Sprintf("%s..origin/%s", b, b)})
		if err != nil {
			commitsBehind = 0
		}
		remotes, _ := repo.RemoteGetURL(b)
		branches[i] = Branch{
			Name:          b,
			LastUpdatedAt: updatedAt,
			CreatedAt:     updatedAt,
			IsCheckedOut:  isHead,
			Remotes:       remotes,
			LastCommitMsg: lastCommitMsg,
			CommitsAhead:  int(commitsAhead),
			CommitsBehind: int(commitsBehind),
		}
	}
	sort.Slice(branches, func(i, j int) bool {
		if branches[j].LastUpdatedAt == nil || branches[i].LastUpdatedAt == nil {
			return false
		}
		return branches[i].LastUpdatedAt.After(*branches[j].LastUpdatedAt)
	})

	headBranch, err := repo.SymbolicRef()
	if err != nil {
		return nil, err
	}
	headBranch, _ = strings.CutPrefix(headBranch, gitm.RefsHeads)

	remotes, err := repo.Remotes(gitm.RemotesOptions{CommandOptions: gitm.CommandOptions{Args: []string{"show"}}})
	if err != nil {
		return nil, err
	}
	origin, err := gitm.RemoteGetURL(dir, "origin", gitm.RemoteGetURLOptions{All: true})
	if err != nil {
		return nil, err
	}

	return &Repo{Repository: *repo, Origin: origin[0], Remotes: remotes, HeadBranchName: headBranch, Branches: branches, Status: status}, nil
}

func GetStatus(dir string) (gitm.NameStatus, error) {
	repo, err := gitm.Open(dir)
	if err != nil {
		return gitm.NameStatus{}, err
	}
	return getUnstagedStatus(repo)
}

// test
func getUnstagedStatus(repo *gitm.Repository) (gitm.NameStatus, error) {
	cmd := gitm.NewCommand("diff", "HEAD", "--name-status")
	stdout, err := cmd.RunInDir(repo.Path())
	if err != nil {
		return gitm.NameStatus{}, err
	}
	status := gitm.NameStatus{}
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		switch fields[0][0] {
		case 'A':
			status.Added = append(status.Added, fields[1])
		case 'D':
			status.Removed = append(status.Removed, fields[1])
		case 'M':
			status.Modified = append(status.Modified, fields[1])
		}
	}
	return status, err
}

func FetchRepo(dir string) (*Repo, error) {
	repo, err := gitm.Open(dir)
	if err != nil {
		return nil, err
	}
	err = repo.Fetch(gitm.FetchOptions{CommandOptions: gitm.CommandOptions{Args: []string{"--all"}}})
	if err != nil {
		return nil, err
	}
	return GetRepo(dir)
}

func GetRepoInPwd() (*gitm.Repository, error) {
	return gitm.Open(".")
}

func GetRepoShortName(url string) string {
	r, _ := strings.CutPrefix(url, "https://github.com/")
	r, _ = strings.CutSuffix(r, ".git")
	return r
}

func GetDiff(repo *Repo, branch string) ([]*gitm.DiffFile, error) {
	err := repo.Repository.Fetch(gitm.FetchOptions{CommandOptions: gitm.CommandOptions{Args: []string{repo.Origin, "main"}}})
	if err != nil {
		return nil, err
	}

	mainBranchCurrentCommit, err := repo.Repository.BranchCommitID("main")
	if err != nil {
		return nil, err
	}

	diff, err := repo.Diff(1000, 1, 1000, gitm.DiffOptions{Base: mainBranchCurrentCommit})
	if err != nil {
		return nil, err
	}

	return diff.Files, nil
}

func (r *Repo) Diff(maxFiles, maxFileLines, maxLineChars int, opts ...gitm.DiffOptions) (*gitm.Diff, error) {
	var opt gitm.DiffOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	cmd := gitm.NewCommand()
	cmd = cmd.AddArgs("diff").
		AddOptions(opt.CommandOptions).
		AddArgs("--full-index", "-M", opt.Base)

	stdout, w := io.Pipe()
	done := make(chan gitm.SteamParseDiffResult)
	go gitm.StreamParseDiff(stdout, done, maxFiles, maxFileLines, maxLineChars)

	stderr := new(bytes.Buffer)
	err := cmd.RunInDirPipeline(w, stderr, r.Repository.Path())
	_ = w.Close() // Close writer to exit parsing goroutine
	if err != nil {
		return nil, concatenateError(err, stderr.String())
	}

	result := <-done
	return result.Diff, result.Err
}

func concatenateError(err error, stderr string) error {
	if len(stderr) == 0 {
		return err
	}
	return fmt.Errorf("%v - %s", err, stderr)
}
