package kaniko

import (
	"regexp"
	"strings"
)

type GitContext struct {
	Repo     string
	Commit   string
	Username string
	Password string
}

func GetGitBuildContext(g *GitContext) (string, error) {
	bCtx := ""

	// cleaning the repo url
	rgx, err := regexp.Compile(`^(https?|git|ssh|ftp)://`)
	if err != nil {
		return "", err
	}
	g.Repo = rgx.ReplaceAllString(g.Repo, "")

	rgx, err = regexp.Compile(`\.git$`)
	if err != nil {
		return "", err
	}
	g.Repo = rgx.ReplaceAllString(g.Repo, "")
	g.Repo = strings.TrimSuffix(g.Repo, "/")

	bCtx += "git://"
	if g.Username != "" {
		bCtx += g.Username
		if g.Password != "" {
			bCtx += ":" + g.Password
		}
		bCtx += "@"
	}

	bCtx += g.Repo
	if g.Commit != "" {
		bCtx += "#" + g.Commit
	}

	return bCtx, nil
}
