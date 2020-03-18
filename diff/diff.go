package diff

import (
	"bytes"
	"fmt"
	"strings"
)

// NOTE: types are code-generated in diff.pb.go.
// First:
//   go mod vendor
//   go install github.com/golang/protobuf/protoc-gen-go
//go:generate protoc --plugin=protoc-gen-gogo=$GOPATH/bin/protoc-gen-go -I=../vendor -I. --gogo_out=. diff.proto

// NewStat creates a blank Stat with internal structures defined
func NewStat() Stat {
	return Stat{
		AddedLineIntervals:   make([]*Stat_LineInterval, 0),
		DeletedLineIntervals: make([]*Stat_LineInterval, 0),
	}
}

// Stat computes the number of lines added/changed/deleted in all
// hunks in this file's diff.
func (d *FileDiff) Stat() Stat {
	total := NewStat()
	for _, h := range d.Hunks {
		total.add(h.Stat())
	}
	return total
}

type statIntervalState struct {
	current *Stat_LineInterval
}

func newStatIntervalState() *statIntervalState {
	return &statIntervalState{
		current: nil,
	}
}

func (s *statIntervalState) updateLine(i int32) {
	if !s.pending() {
		s.current = &Stat_LineInterval{Start: i}
	}
	s.current.End = i
}

func (s *statIntervalState) pending() bool {
	return s.current != nil
}

func (s *statIntervalState) popInterval() (interval *Stat_LineInterval) {
	interval = s.current
	s.current = nil
	return
}

func (s *statIntervalState) appendInterval(sli *[]*Stat_LineInterval) {
	if s.pending() {
		*sli = append(*sli, s.popInterval())
	}
}

// Stat computes the number of lines added/changed/deleted in this
// hunk.
func (h *Hunk) Stat() Stat {
	lines := bytes.Split(h.Body, []byte{'\n'})
	var last byte
	var lastFlag byte

	st := NewStat()
	addedState := newStatIntervalState()
	deletedState := newStatIntervalState()

	for lineNbr, line := range lines {
		if len(line) == 0 {
			last, lastFlag = 0, 0
			continue
		}

		if line[0] == '+' {
			lineInNewFile := int32(h.NewStartLine) + int32(lineNbr) - st.Deleted - st.Changed
			addedState.updateLine(lineInNewFile)
		} else {
			if lastFlag == '+' {
				addedState.appendInterval(&st.AddedLineIntervals)
			}
		}

		if line[0] == '-' {
			lineInOrigFile := int32(h.OrigStartLine) + int32(lineNbr) - st.Added - st.Changed
			deletedState.updateLine(lineInOrigFile)
		} else {
			if lastFlag == '-' {
				deletedState.appendInterval(&st.DeletedLineIntervals)
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

		lastFlag = line[0]
	}

	addedState.appendInterval(&st.AddedLineIntervals)
	deletedState.appendInterval(&st.DeletedLineIntervals)

	return st
}

// FormatAddedLineIntervals renders the added intevals as a list of ranges (start-end) or single value
func (m *Stat) FormatAddedLineIntervals() string {
	return intervalsString(m.AddedLineIntervals)
}

// FormatDeletedLineIntervals renders the deleted intevals as a list of ranges (start-end) or single value
func (m *Stat) FormatDeletedLineIntervals() string {
	return intervalsString(m.DeletedLineIntervals)
}

func intervalsString(sli []*Stat_LineInterval) string {
	var b strings.Builder
	// estimate the needed buffer size, to avoid excessive reallocations
	b.Grow(len(sli) * 10)
	for i, pair := range sli {
		if pair.Start == pair.End {
			fmt.Fprintf(&b, "%d", pair.Start)
		} else {
			fmt.Fprintf(&b, "%d-%d", pair.Start, pair.End)
		}

		if i != len(sli)-1 {
			fmt.Fprint(&b, ", ")
		}
	}
	return b.String()
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
