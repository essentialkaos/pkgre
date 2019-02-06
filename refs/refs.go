// Git refs parser
package refs

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2019 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type RefType uint8

type Info struct {
	branches map[string]string // branch -> rev
	tags     map[string]string // tag -> rev
	raw      []byte
}

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	TYPE_UNKNOWN RefType = iota
	TYPE_BRANCH
	TYPE_TAG
)

// ////////////////////////////////////////////////////////////////////////////////// //

// TagList returns slice of tag names
func (r *Info) TagList() []string {
	if r == nil {
		return []string{}
	}

	var result []string

	for t := range r.tags {
		result = append(result, t)
	}

	return result
}

// BranchList returns slice of branch names
func (r *Info) BranchList() []string {
	if r == nil {
		return []string{}
	}

	var result []string

	for b := range r.branches {
		result = append(result, b)
	}

	return result
}

// HasBranch returns true if branch with given name is exist in repo
func (r *Info) HasBranch(name string) bool {
	if r == nil || r.branches == nil {
		return false
	}

	return r.branches[name] != ""
}

// HasBranch returns true if tag with given name is exist in repo
func (r *Info) HasTag(name string) bool {
	if r == nil || r.tags == nil {
		return false
	}

	return r.tags[name] != ""
}

// GetTagSHA returns SHA for given tag
func (r *Info) GetTagSHA(name string, short bool) string {
	if r == nil || r.tags == nil {
		return ""
	}

	return formatSHA(r.tags[name], short)
}

// GetBranchSHA returns SHA for given branch
func (r *Info) GetBranchSHA(name string, short bool) string {
	if r == nil || r.branches == nil {
		return ""
	}

	return formatSHA(r.branches[name], short)
}

// Rewrite returns refs with updated head
func (r *Info) Rewrite(headName string, headType RefType) []byte {
	// If head name is empty we return unchanged refs
	if headName == "" {
		return r.raw
	}

	var (
		rBuf    *bytes.Buffer
		wBuf    bytes.Buffer
		refName string
		refSHA  string
	)

	rBuf = bytes.NewBuffer(r.raw)
	wBuf.Grow(len(r.raw) + 256)

	switch headType {
	case TYPE_TAG:
		refName = "refs/tags/" + headName
		refSHA = r.tags[headName]
	case TYPE_BRANCH:
		refName = "refs/heads/" + headName
		refSHA = r.branches[headName]
	}

	var lines int

	for {
		lines++

		line, err := rBuf.ReadString('\n')

		if err == io.EOF {
			fmt.Fprintln(&wBuf, line)
			break
		}

		line = line[:len(line)-1]

		if lines == 2 {
			fmt.Fprintf(&wBuf, rewriteHeadRefs(line, refName, refSHA))
			continue
		}

		if strings.HasSuffix(line, "refs/heads/master") {
			refLine := refSHA + " " + "refs/heads/master\n"
			fmt.Fprintf(&wBuf, "%04x%s", 4+len(refLine), refLine)
		} else {
			fmt.Fprintln(&wBuf, line)
		}
	}

	return wBuf.Bytes()
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Parse parse data and return refs struct and error
func Parse(data []byte) (*Info, error) {
	refs := &Info{
		branches: make(map[string]string),
		tags:     make(map[string]string),
		raw:      data,
	}

	var lines int

	buf := bytes.NewBuffer(data)

	for {
		lines++

		if lines == 1 {
			continue
		}

		line, err := buf.ReadString('\n')

		if err == io.EOF || line == "0000\n" {
			break
		}

		line = line[:len(line)-1]

		typ, name, sha := parseRefLine(line)

		switch typ {
		case TYPE_BRANCH:
			refs.branches[name] = sha
		case TYPE_TAG:
			refs.tags[name] = sha
		}
	}

	if lines <= 3 {
		return nil, errors.New("Refs data is malfomed")
	}

	return refs, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// parseRefLine parse line with refs and return type, name and hash
func parseRefLine(data string) (RefType, string, string) {
	if len(data) < 55 {
		return TYPE_UNKNOWN, "", ""
	}

	sha := data[4:44]
	name := data[45:]

	if strings.HasSuffix(name, "^{}") {
		name = name[0 : len(name)-3]
	}

	if name[:10] == "refs/tags/" {
		return TYPE_TAG, name[10:], sha
	} else if name[:11] == "refs/heads/" {
		return TYPE_BRANCH, name[11:], sha
	} else {
		return TYPE_UNKNOWN, "", ""
	}
}

// formatSHA return formated (short/long) SHA hash
func formatSHA(sha string, short bool) string {
	if len(sha) < 8 {
		return ""
	}

	switch short {
	case true:
		return sha[:8]
	default:
		return sha
	}
}

// rewriteHeadRefs return head line with new head refs
func rewriteHeadRefs(head, refName, refSHA string) string {
	var result []string

	headSlice := strings.Split(head, " ")

	for index, headPart := range headSlice {
		if index == 0 {
			result = append(result, refSHA)
			continue
		}

		if strings.HasPrefix(headPart, "symref=") {
			if strings.HasPrefix(refName, "refs/heads/") {
				result = append(result, "symref=HEAD:"+refName)
			} else {
				result = append(result, headPart)
			}

			result = append(result, "oldref="+headPart[7:])
			continue
		}

		result = append(result, headPart)
	}

	headLine := strings.Join(result, " ") + "\n"

	return fmt.Sprintf("0000%04x%s", 4+len(headLine), headLine)
}
