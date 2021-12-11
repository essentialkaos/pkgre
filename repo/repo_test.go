package repo

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2021 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"testing"

	. "pkg.re/essentialkaos/check.v1"
)

// ////////////////////////////////////////////////////////////////////////////////// //

func Test(t *testing.T) { TestingT(t) }

// ////////////////////////////////////////////////////////////////////////////////// //

type RepoSuite struct{}

// ////////////////////////////////////////////////////////////////////////////////// //

var _ = Suite(&RepoSuite{})

// ////////////////////////////////////////////////////////////////////////////////// //

func (s *RepoSuite) TestParsePath(c *C) {
	info, err := ParsePath("/essentialkaos/ek.v12")

	c.Assert(info, NotNil)
	c.Assert(err, IsNil)
	c.Assert(info.User, Equals, "essentialkaos")
	c.Assert(info.Name, Equals, "ek")
	c.Assert(info.Path, Equals, "")
	c.Assert(info.Target, Equals, "v12")
	c.Assert(info.Validate(), IsNil)

	info, err = ParsePath("/essentialkaos/ek.v12.34.1/knf/validators/regexp")

	c.Assert(info, NotNil)
	c.Assert(err, IsNil)
	c.Assert(info.User, Equals, "essentialkaos")
	c.Assert(info.Name, Equals, "ek")
	c.Assert(info.Path, Equals, "knf/validators/regexp")
	c.Assert(info.Target, Equals, "v12.34.1")
	c.Assert(info.Validate(), IsNil)

	info, err = ParsePath("/essentialkaos/ek.v12.git")

	c.Assert(info, NotNil)
	c.Assert(err, IsNil)
	c.Assert(info.User, Equals, "essentialkaos")
	c.Assert(info.Name, Equals, "ek")
	c.Assert(info.Path, Equals, "")
	c.Assert(info.Target, Equals, "v12")
	c.Assert(info.Validate(), IsNil)

	info, err = ParsePath("/or-ga-ni-za-tion-6/mySupper_REPO.v12.0.1/a/b/c/d")

	c.Assert(info, NotNil)
	c.Assert(err, IsNil)
	c.Assert(info.User, Equals, "or-ga-ni-za-tion-6")
	c.Assert(info.Name, Equals, "mySupper_REPO")
	c.Assert(info.Path, Equals, "a/b/c/d")
	c.Assert(info.Target, Equals, "v12.0.1")
	c.Assert(info.Validate(), IsNil)

	info, err = ParsePath("/yaml.v5/parser")

	c.Assert(info, NotNil)
	c.Assert(err, IsNil)
	c.Assert(info.User, Equals, "go-yaml")
	c.Assert(info.Name, Equals, "yaml")
	c.Assert(info.Path, Equals, "parser")
	c.Assert(info.Target, Equals, "v5")
	c.Assert(info.Validate(), IsNil)

	info, err = ParsePath("/yaml")

	c.Assert(info, NotNil)
	c.Assert(err, IsNil)
	c.Assert(info.User, Equals, "go-yaml")
	c.Assert(info.Name, Equals, "yaml")
	c.Assert(info.Path, Equals, "")
	c.Assert(info.Target, Equals, "")
	c.Assert(info.Validate(), IsNil)
}

func (s *RepoSuite) TestInfoValidator(c *C) {
	info := &Info{User: "", Name: "", Path: "", Target: ""}
	c.Assert(info.Validate(), DeepEquals, ErrInvalidUser)

	info = &Info{User: ".john", Name: "", Path: "", Target: ""}
	c.Assert(info.Validate(), DeepEquals, ErrInvalidUser)

	info = &Info{User: "john", Name: "", Path: "", Target: ""}
	c.Assert(info.Validate(), DeepEquals, ErrInvalidName)

	info = &Info{User: "john", Name: "1+1", Path: "", Target: ""}
	c.Assert(info.Validate(), DeepEquals, ErrInvalidName)

	info = &Info{User: "john", Name: "test", Path: "/++++", Target: ""}
	c.Assert(info.Validate(), DeepEquals, ErrInvalidPath)
}

func (s *RepoSuite) TestParsePathErrors(c *C) {
	_, err := ParsePath("")
	c.Assert(err, DeepEquals, ErrUnsupportedURL)

	_, err = ParsePath("test")
	c.Assert(err, DeepEquals, ErrUnsupportedURL)
}

func (s *RepoSuite) TestHelpers(c *C) {
	info, err := ParsePath("/essentialkaos/ek.v12.36.0/pid")

	c.Assert(info, NotNil)
	c.Assert(err, IsNil)

	c.Assert(info.GitHubRoot(), Equals, "github.com/essentialkaos/ek")
	c.Assert(info.FullPath(), Equals, "essentialkaos/ek.v12.36.0/pid")
	c.Assert(info.GitHubURL("v12.36.0"), Equals, "https://github.com/essentialkaos/ek/tree/v12.36.0/pid")
	c.Assert(info.GitHubURL(""), Equals, "https://github.com/essentialkaos/ek")

	info = &Info{Name: "test"}
	c.Assert(info.Root(), Equals, "test")
	info = &Info{Name: "test", User: "john"}
	c.Assert(info.Root(), Equals, "john/test")
	info = &Info{Name: "test", User: "john", Target: "v1.2.3"}
	c.Assert(info.Root(), Equals, "john/test.v1.2.3")
	c.Assert(info.FullPath(), Equals, "john/test.v1.2.3")
}

// ////////////////////////////////////////////////////////////////////////////////// //

func (s *RepoSuite) BenchmarkParsePath(c *C) {
	for i := 0; i < c.N; i++ {
		ParsePath("/essentialkaos/ek.v12")
	}
}
