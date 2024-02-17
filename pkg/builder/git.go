package builder

import (
	"regexp"
	"strings"
)

const (
	regexpGitRepoProtocol = `^(https?|git|ssh|ftp)://`
	regexpGitRepoDotGit   = `\.git$`
	gitProtocol           = "git://"
)

type GitContext struct {
	Repo     string
	Branch   string
	Commit   string
	Username string
	Password string
}

// This build context follows Kaniko build context pattern
// ref: https://github.com/GoogleContainerTools/kaniko#kaniko-build-contexts
func (g *GitContext) BuildContext() (string, error) {
	bCtx := ""

	// cleaning the repo url
	rgx, err := regexp.Compile(regexpGitRepoProtocol)
	if err != nil {
		return "", err
	}
	g.Repo = rgx.ReplaceAllString(g.Repo, "")

	rgx, err = regexp.Compile(regexpGitRepoDotGit)
	if err != nil {
		return "", err
	}
	g.Repo = rgx.ReplaceAllString(g.Repo, "")
	g.Repo = strings.TrimSuffix(g.Repo, "/")

	bCtx += gitProtocol
	if g.Username != "" {
		bCtx += g.Username
		if g.Password != "" {
			bCtx += ":" + g.Password
		}
		bCtx += "@"
	}

	bCtx += g.Repo
	if g.Branch != "" {
		bCtx += "#refs/heads/" + g.Branch
	}

	if g.Commit != "" {
		bCtx += "#" + g.Commit
	}

	return bCtx, nil
}

func IsGitContext(ctx string) bool {
	return strings.HasPrefix(ctx, gitProtocol)
}
