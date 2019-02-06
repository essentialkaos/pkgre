package refs

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2019 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bytes"
	"io/ioutil"
	"testing"

	. "pkg.re/check.v1"
)

// ////////////////////////////////////////////////////////////////////////////////// //

func Test(t *testing.T) { TestingT(t) }

// ////////////////////////////////////////////////////////////////////////////////// //

type RefsSuite struct{}

// ////////////////////////////////////////////////////////////////////////////////// //

var _ = Suite(&RefsSuite{})

// ////////////////////////////////////////////////////////////////////////////////// //

func (s *RefsSuite) TestBasicParsing(c *C) {
	data, err := ioutil.ReadFile("../testdata/refs.dat")

	if err != nil {
		c.Fatal(err.Error())
	}

	info, err := Parse(data)

	c.Assert(err, IsNil)
	c.Assert(info, NotNil)

	c.Assert(info.TagList(), HasLen, 10)
	c.Assert(info.BranchList(), HasLen, 2)

	var nullInfo *Info

	c.Assert(nullInfo.TagList(), HasLen, 0)
	c.Assert(nullInfo.BranchList(), HasLen, 0)

	c.Assert(info.HasBranch("master"), Equals, true)
	c.Assert(info.HasBranch("unknown"), Equals, false)
	c.Assert(info.GetBranchSHA("master", true), Equals, "3e4111e9")
	c.Assert(info.GetBranchSHA("master", false), Equals, "3e4111e9efcaa0e16a652589c75dc98910a79cab")
	c.Assert(nullInfo.HasBranch("unknown"), Equals, false)
	c.Assert(nullInfo.GetBranchSHA("master", true), Equals, "")

	c.Assert(info.HasTag("v3.6.0"), Equals, true)
	c.Assert(info.HasTag("v0.0.0"), Equals, false)
	c.Assert(info.GetTagSHA("v3.6.0", true), Equals, "c766ee99")
	c.Assert(info.GetTagSHA("v3.6.0", false), Equals, "c766ee99f84d21dbd9cceb1ecbc5a6dae956efef")
	c.Assert(nullInfo.HasTag("v0.0.0"), Equals, false)
	c.Assert(nullInfo.GetTagSHA("v3.6.0", true), Equals, "")

	info, err = Parse([]byte("abc\n"))

	c.Assert(err, NotNil)
	c.Assert(info, IsNil)
}

func (s *RefsSuite) TestRewrite(c *C) {
	data, err := ioutil.ReadFile("../testdata/refs.dat")

	if err != nil {
		c.Fatal(err.Error())
	}

	info, err := Parse(data)

	c.Assert(err, IsNil)
	c.Assert(info, NotNil)

	newData := info.Rewrite("", TYPE_BRANCH)

	c.Assert(bytes.Contains(newData, []byte("3e4111e9efcaa0e16a652589c75dc98910a79cab HEAD")), Equals, true)

	newData = info.Rewrite("develop", TYPE_BRANCH)

	c.Assert(bytes.Contains(newData, []byte("daa684d3e025e542e542472df3905fb26e41fc60 HEAD")), Equals, true)
	c.Assert(bytes.Contains(newData, []byte("symref=HEAD:refs/heads/develop oldref=HEAD:refs/heads/master")), Equals, true)

	newData = info.Rewrite("v3.6.0", TYPE_TAG)

	c.Assert(bytes.Contains(newData, []byte("c766ee99f84d21dbd9cceb1ecbc5a6dae956efef HEAD")), Equals, true)
}

func (s *RefsSuite) TestSHAFormat(c *C) {
	sha := "3e4111e9efcaa0e16a652589c75dc98910a79cab"

	c.Assert(formatSHA(sha, true), Equals, "3e4111e9")
	c.Assert(formatSHA(sha, false), Equals, sha)
	c.Assert(formatSHA("", false), Equals, "")
}

// ////////////////////////////////////////////////////////////////////////////////// //

func (s *RefsSuite) BenchmarkParsing(c *C) {
	data, err := ioutil.ReadFile("../testdata/refs.dat")

	if err != nil {
		c.Fatal(err.Error())
	}

	for i := 0; i < c.N; i++ {
		Parse(data)
	}
}
