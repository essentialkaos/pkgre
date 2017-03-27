// Git refs parser
package refs

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2017 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type RefType uint8

type Info struct {
	branches map[string]string // branch -> rev
	tags     map[string]string // tag -> rev
	raw      []string
	rawSize  int
}

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	TYPE_UNKNOWN RefType = iota
	TYPE_BRANCH
	TYPE_TAG
)

// ////////////////////////////////////////////////////////////////////////////////// //

// TagList return slice of tag names
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

// BranchList return slice of branch names
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

// HasBranch return true if branch with given name is exist in repo
func (r *Info) HasBranch(name string) bool {
	return r.branches[name] != ""
}

// HasBranch return true if tag with given name is exist in repo
func (r *Info) HasTag(name string) bool {
	return r.tags[name] != ""
}

// GetTagSHA return SHA for given tag
func (r *Info) GetTagSHA(name string, short bool) string {
	sha := r.tags[name]

	if sha == "" {
		return ""
	}

	return formatSHA(sha, short)
}

// GetBranchSHA return SHA for given branch
func (r *Info) GetBranchSHA(name string, short bool) string {
	sha := r.branches[name]

	if sha == "" {
		return ""
	}

	return formatSHA(sha, short)
}

// Rewrite return refs with updated head
func (r *Info) Rewrite(headName string, headType RefType) []byte {
	// If head name is empty we return unchanged refs
	if headName == "" {
		return []byte(strings.Join(r.raw, "\n"))
	}

	var (
		buf     bytes.Buffer
		refName string
		refSHA  string
	)

	buf.Grow(r.rawSize + 256)

	switch headType {
	case TYPE_TAG:
		refName = "refs/tags/" + headName
		refSHA = r.tags[headName]
	case TYPE_BRANCH:
		refName = "refs/heads/" + headName
		refSHA = r.branches[headName]
	}

	lastLine := len(r.raw) - 1

	for index, line := range r.raw {
		switch index {
		case 1:
			fmt.Fprintf(&buf, rewriteHeadRefs(line, refName, refSHA))
		case lastLine:
			fmt.Fprintf(&buf, line)
		default:
			if strings.HasSuffix(line, "refs/heads/master") {
				refLine := refSHA + " " + "refs/heads/master\n"
				fmt.Fprintf(&buf, "%04x%s", 4+len(refLine), refLine)
			} else {
				fmt.Fprintln(&buf, line)
			}
		}
	}

	return buf.Bytes()
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Parse parse data and return refs struct and error
func Parse(data []byte) (*Info, error) {
	strDataSlice := strings.Split(string(data[:]), "\n")

	// Well formated refs data must contains 3 or more lines
	if len(strDataSlice) <= 3 {
		return nil, errors.New("Info data is malfomed")
	}

	refs := &Info{
		branches: make(map[string]string),
		tags:     make(map[string]string),
		rawSize:  len(data),
		raw:      strDataSlice,
	}

	for lineNum, line := range strDataSlice {
		switch lineNum {
		case 0:
			continue
		}

		// End of refs
		if line == "0000" {
			break
		}

		t, name, sha := parseRefLine(line)

		switch t {
		case TYPE_BRANCH:
			refs.branches[name] = sha
		case TYPE_TAG:
			refs.tags[name] = sha
		}
	}

	return refs, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// parseRefLine parse line with refs and return type, name and hash
func parseRefLine(data string) (RefType, string, string) {
	lineSlice := strings.Split(data, " ")

	if len(lineSlice) < 2 {
		return TYPE_UNKNOWN, "", ""
	}

	name, sha := lineSlice[1], lineSlice[0]

	if strings.HasSuffix(name, "^{}") {
		name = name[0 : len(name)-3]
	}

	if name[:10] == "refs/tags/" {
		return TYPE_TAG, name[10:], sha[4:]
	} else if name[:11] == "refs/heads/" {
		return TYPE_BRANCH, name[11:], sha[4:]
	} else {
		return TYPE_UNKNOWN, "", ""
	}
}

// safeLineParse split line
func safeLineParse(data string, index int) string {
	if data == "" {
		return ""
	}

	slice := strings.Split(data, " ")

	if len(slice) < index {
		return ""
	}

	return slice[index]
}

// formatSHA return formated (short/long) SHA hash
func formatSHA(sha string, short bool) string {
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
			}

			result = append(result, "oldref="+headPart[7:])
			continue
		}

		result = append(result, headPart)
	}

	headLine := strings.Join(result, " ") + "\n"

	return fmt.Sprintf("0000%04x%s", 4+len(headLine), headLine)
}
