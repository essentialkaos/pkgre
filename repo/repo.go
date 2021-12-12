package repo

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2021 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"errors"
	"regexp"
	"strings"

	"pkg.re/essentialkaos/ek.v12/strutil"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Info contains basic information about repository
type Info struct {
	User   string
	Name   string
	Path   string
	Target string
}

// ////////////////////////////////////////////////////////////////////////////////// //

var (
	userValidationRegExp = regexp.MustCompile(`^[a-zA-Z0-9][\w\d_\-]+$`)
	nameValidationRegExp = regexp.MustCompile(`^[\w\d_.\-]{2,}$`)
	pathValidationRegExp = regexp.MustCompile(`^[\w\d_.\-\/]*$`)
)

var (
	ErrUnsupportedURL = errors.New("Unsupported URL pattern")
	ErrInvalidUser    = errors.New("Repository username is not valid")
	ErrInvalidName    = errors.New("Repository name is not valid")
	ErrInvalidPath    = errors.New("Repository path is not valid")
)

// ////////////////////////////////////////////////////////////////////////////////// //

// ParsePath parses given path to repo struct
func ParsePath(path string) (*Info, error) {
	var repoUser, repoName, repoTarget, repoPath string

	if strings.HasSuffix(path, ".git") {
		path = path[:len(path)-4]
	}

	if len(path) == 0 || path[0] != '/' || strings.Count(path, "/") == 0 {
		return nil, ErrUnsupportedURL
	}

	// Remove leading slash
	path = path[1:]

	repoUser = strutil.ReadField(path, 0, false, "/")

	// Check short notation (pkg.re/mgo or pkg.re/mgo.v1)
	if strings.ContainsRune(repoUser, '.') || strings.Count(path, "/") == 0 {
		repoPath = strutil.Exclude(path, repoUser)
		repoName, repoTarget = parseNameAndTarget(repoUser)
		repoUser = "go-" + repoName
	} else {
		repoName = strutil.ReadField(path, 1, false, "/")
		repoPath = strutil.Exclude(path, repoUser+"/"+repoName)
		repoName, repoTarget = parseNameAndTarget(repoName)
	}

	repoPath = strings.TrimLeft(repoPath, "/")

	return &Info{
		User:   repoUser,
		Name:   repoName,
		Path:   repoPath,
		Target: repoTarget,
	}, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Validate validates repository info (user, name and path)
func (i *Info) Validate() error {
	if !userValidationRegExp.MatchString(i.User) {
		return ErrInvalidUser
	}

	if !nameValidationRegExp.MatchString(i.Name) {
		return ErrInvalidName
	}

	if i.Path != "" {
		if !pathValidationRegExp.MatchString(i.Path) {
			return ErrInvalidPath
		}
	}

	return nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// GitHubRoot returns GitHub root path e.g. github.com/user/project
func (i *Info) GitHubRoot() string {
	return "github.com/" + i.User + "/" + i.Name
}

// GitHubURL returns URL of repository on github
func (i *Info) GitHubURL(branchOrTag string) string {
	if branchOrTag == "" {
		return "https://" + i.GitHubRoot()
	}

	url := "https://" + i.GitHubRoot() + "/tree/" + branchOrTag

	if i.Path != "" {
		url += "/" + i.Path
	}

	return url
}

// Root returns root path for some repo e.g. user/project.target
func (i *Info) Root() string {
	var target = ""

	if i.Target != "" {
		target = "." + i.Target
	}

	if i.User == "" {
		return i.Name + target
	}

	return i.User + "/" + i.Name + target
}

// FullPath returns full path e.g. user/project.target/some/part
func (i *Info) FullPath() string {
	if i.Path != "" {
		return i.Root() + "/" + i.Path
	}

	return i.Root()
}

// ////////////////////////////////////////////////////////////////////////////////// //

func parseNameAndTarget(name string) (string, string) {
	if !strings.ContainsRune(name, '.') {
		return name, ""
	}

	separator := strings.IndexRune(name, '.')

	return name[:separator], name[separator+1:]
}
