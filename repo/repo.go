package repo

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2020 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"errors"
	"regexp"
	"strings"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Info contains basic information about repository
type Info struct {
	User   string
	Name   string
	Path   string
	Target string
}

var userValidationRegExp = regexp.MustCompile(`[A-Za-z0-9_-]{2,}`)
var nameValidationRegExp = regexp.MustCompile(`[A-Za-z0-9_.-]{2,}`)
var pathValidationRegExp = regexp.MustCompile(`[A-Za-z0-9_.-/]{0,}`)

// ////////////////////////////////////////////////////////////////////////////////// //

// ParsePath parses given path to repo struct
func ParsePath(path string) (*Info, error) {
	if strings.Contains(path, ".git") {
		path = strings.Replace(path, ".git", "", -1)
	}

	pathSlice := strings.Split(path, "/")

	if len(pathSlice) <= 1 {
		return nil, errors.New("Unsupported URL pattern")
	}

	var (
		repoUser   string
		repoName   string
		repoTarget string
		repoPath   string
	)

	// Check short notation (pkg.re/mgo or pkg.re/mgo.v1)
	if strings.Contains(pathSlice[1], ".") || len(pathSlice) == 2 {
		repoUser = ""
		repoName = pathSlice[1]

		if len(pathSlice) > 2 {
			repoPath = strings.Join(pathSlice[2:], "/")
		}
	} else {
		repoUser = pathSlice[1]
		repoName = pathSlice[2]

		if len(pathSlice) > 3 {
			repoPath = strings.Join(pathSlice[3:], "/")
		}
	}

	dotIndex := strings.Index(repoName, ".")

	if dotIndex != -1 {
		repoTarget = repoName[dotIndex+1:]
		repoName = repoName[:dotIndex]
	}

	if repoUser != "" {
		if !userValidationRegExp.MatchString(repoUser) {
			return nil, errors.New("Repo username is not valid")
		}
	}

	if repoName != "" {
		if !nameValidationRegExp.MatchString(repoName) {
			return nil, errors.New("Repo name is not valid")
		}
	}

	if repoPath != "" {
		if !pathValidationRegExp.MatchString(repoPath) {
			return nil, errors.New("Repo sub-path is not valid")
		}
	}

	return &Info{
		User:   repoUser,
		Name:   repoName,
		Path:   repoPath,
		Target: repoTarget,
	}, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// GitHubRoot return github root path e.g. github.com/user/project
func (i *Info) GitHubRoot() string {
	if i.User == "" {
		return "github.com/go-" + i.Name + "/" + i.Name
	}

	return "github.com/" + i.User + "/" + i.Name
}

// GitHubURL return url of repository on github
func (i *Info) GitHubURL(branchOrName string) string {
	url := "https://" + i.GitHubRoot()

	if branchOrName != "" {
		url += "/tree/" + branchOrName
	}

	if i.Path != "" {
		url += "/" + i.Path
	}

	return url
}

// Root return root path for some repo e.g. user/project.target
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

// FullPath return full path e.g. user/project.target/some/part
func (i *Info) FullPath() string {
	if i.Path != "" {
		return i.Root() + "/" + i.Path
	}

	return i.Root()
}
