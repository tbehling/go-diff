package diff

import "bytes"

// NOTE: types are code-generated in diff.pb.go.
// First:
//   go mod vendor
//   go install github.com/golang/protobuf/protoc-gen-go
//go:generate protoc --plugin=protoc-gen-gogo=$GOPATH/bin/protoc-gen-go -I=../vendor -I. --gogo_out=. diff.proto

// Stat computes the number of lines added/changed/deleted in all
// hunks in this file's diff.
func (d *FileDiff) Stat() Stat {
	total := Stat{
		AddedLineIntervals:   make([]*Stat_LineInterval, 0),
		DeletedLineIntervals: make([]*Stat_LineInterval, 0),
	}
	for _, h := range d.Hunks {
		total.add(h.Stat())
	}
	return total
}

// Stat computes the number of lines added/changed/deleted in this
// hunk.
func (h *Hunk) Stat() Stat {
	lines := bytes.Split(h.Body, []byte{'\n'})
	var last byte

	st := Stat{
		AddedLineIntervals:   make([]*Stat_LineInterval, 0),
		DeletedLineIntervals: make([]*Stat_LineInterval, 0),
	}

	addedInterval := &Stat_LineInterval{}
	deletedInterval := &Stat_LineInterval{}
	var lastState byte
	pendingIntervalAdd, pendingIntervalDel := false, false

	for lineNbr, line := range lines {
		if len(line) == 0 {
			last = 0
			continue
		}

		lineInNewFile := int32(h.NewStartLine) + int32(lineNbr) - st.Deleted - st.Changed
		lineInOrigFile := int32(h.OrigStartLine) + int32(lineNbr) - st.Added - st.Changed

		if line[0] == '+' {
			if lastState != '+' {
				addedInterval.Start = lineInNewFile
				pendingIntervalAdd = true
			}
			addedInterval.End = lineInNewFile
		} else {
			if lastState == '+' {
				st.AddedLineIntervals = append(st.AddedLineIntervals, addedInterval)
				pendingIntervalAdd = false
				addedInterval = &Stat_LineInterval{}
			}
		}

		if line[0] == '-' {
			if lastState != '-' {
				deletedInterval.Start = lineInOrigFile
				pendingIntervalDel = true
			}
			deletedInterval.End = lineInOrigFile
		} else {
			if lastState == '-' {
				st.DeletedLineIntervals = append(st.DeletedLineIntervals, deletedInterval)
				pendingIntervalDel = false
				deletedInterval = &Stat_LineInterval{}
			}
		}

		switch line[0] {
		case '-':
			if last == '+' {
				st.Added--
				st.Changed++
				last = 0 // next line can't change this one since this is already a change
			} else {
				st.Deleted++
				last = line[0]
			}
		case '+':
			if last == '-' {
				st.Deleted--
				st.Changed++
				last = 0 // next line can't change this one since this is already a change
			} else {
				st.Added++
				last = line[0]
			}
		default:
			last = 0
		}

		lastState = line[0]
	}

	if pendingIntervalAdd {
		st.AddedLineIntervals = append(st.AddedLineIntervals, addedInterval)
	}
	if pendingIntervalDel {
		st.DeletedLineIntervals = append(st.DeletedLineIntervals, deletedInterval)
	}

	return st
}

var (
	hunkPrefix = []byte("@@ ")
)

const hunkHeader = "@@ -%d,%d +%d,%d @@"

// diffTimeParseLayout is the layout used to parse the time in unified diff file
// header timestamps.
// See https://www.gnu.org/software/diffutils/manual/html_node/Detailed-Unified.html.
const diffTimeParseLayout = "2006-01-02 15:04:05 -0700"

// diffTimeFormatLayout is the layout used to format (i.e., print) the time in unified diff file
// header timestamps.
// See https://www.gnu.org/software/diffutils/manual/html_node/Detailed-Unified.html.
const diffTimeFormatLayout = "2006-01-02 15:04:05.000000000 -0700"

func (s *Stat) add(o Stat) {
	s.Added += o.Added
	s.Changed += o.Changed
	s.Deleted += o.Deleted

	s.AddedLineIntervals = append(s.AddedLineIntervals, o.AddedLineIntervals...)
	s.DeletedLineIntervals = append(s.DeletedLineIntervals, o.DeletedLineIntervals...)
}
