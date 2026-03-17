package service

import (
	"fmt"
	"strings"
)

// unifiedDiff generates a unified diff between two sets of lines.
func unifiedDiff(a, b []string, labelA, labelB string) string {
	ops := computeEditScript(a, b)
	hunks := groupHunks(ops, 3)

	if len(hunks) == 0 {
		return ""
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "--- %s\n", labelA)
	fmt.Fprintf(&buf, "+++ %s\n", labelB)

	for _, h := range hunks {
		fmt.Fprintf(&buf, "@@ -%d,%d +%d,%d @@\n", h.aStart+1, h.aCount, h.bStart+1, h.bCount)
		for _, line := range h.lines {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}

	return buf.String()
}

type editOp int

const (
	opEqual  editOp = iota
	opDelete        // line in a only
	opInsert        // line in b only
)

type edit struct {
	op   editOp
	text string
	aIdx int // line index in a (-1 for insert)
	bIdx int // line index in b (-1 for delete)
}

// computeEditScript uses LCS to produce an edit script.
func computeEditScript(a, b []string) []edit {
	m, n := len(a), len(b)

	// LCS table.
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to build edit script (reversed).
	var ops []edit
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			ops = append(ops, edit{op: opEqual, text: a[i-1], aIdx: i - 1, bIdx: j - 1})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, edit{op: opInsert, text: b[j-1], aIdx: -1, bIdx: j - 1})
			j--
		} else {
			ops = append(ops, edit{op: opDelete, text: a[i-1], aIdx: i - 1, bIdx: -1})
			i--
		}
	}

	// Reverse to get forward order.
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}

	return ops
}

type hunk struct {
	aStart int
	aCount int
	bStart int
	bCount int
	lines  []string
}

// groupHunks collects edits into unified diff hunks with the given context.
func groupHunks(ops []edit, context int) []hunk {
	// Find change regions (non-equal edits).
	type region struct{ start, end int }
	var regions []region
	for i, op := range ops {
		if op.op != opEqual {
			if len(regions) == 0 || i > regions[len(regions)-1].end+1 {
				regions = append(regions, region{i, i})
			} else {
				regions[len(regions)-1].end = i
			}
		}
	}

	if len(regions) == 0 {
		return nil
	}

	// Merge nearby regions based on context overlap.
	var merged []region
	for _, r := range regions {
		start := r.start - context
		if start < 0 {
			start = 0
		}
		end := r.end + context
		if end >= len(ops) {
			end = len(ops) - 1
		}

		if len(merged) > 0 && start <= merged[len(merged)-1].end+1 {
			merged[len(merged)-1].end = end
		} else {
			merged = append(merged, region{start, end})
		}
	}

	// Build hunks.
	var hunks []hunk
	for _, r := range merged {
		h := hunk{}
		aStartSet := false
		bStartSet := false
		for i := r.start; i <= r.end; i++ {
			op := ops[i]
			switch op.op {
			case opEqual:
				if !aStartSet {
					h.aStart = op.aIdx
					aStartSet = true
				}
				if !bStartSet {
					h.bStart = op.bIdx
					bStartSet = true
				}
				h.aCount++
				h.bCount++
				h.lines = append(h.lines, " "+op.text)
			case opDelete:
				if !aStartSet {
					h.aStart = op.aIdx
					aStartSet = true
				}
				if !bStartSet {
					// Find the next b index from following ops.
					for j := i + 1; j < len(ops); j++ {
						if ops[j].bIdx >= 0 {
							h.bStart = ops[j].bIdx
							bStartSet = true
							break
						}
					}
					if !bStartSet {
						h.bStart = h.aStart
						bStartSet = true
					}
				}
				h.aCount++
				h.lines = append(h.lines, "-"+op.text)
			case opInsert:
				if !bStartSet {
					h.bStart = op.bIdx
					bStartSet = true
				}
				if !aStartSet {
					// Find the next a index from following ops.
					for j := i + 1; j < len(ops); j++ {
						if ops[j].aIdx >= 0 {
							h.aStart = ops[j].aIdx
							aStartSet = true
							break
						}
					}
					if !aStartSet {
						h.aStart = h.bStart
						aStartSet = true
					}
				}
				h.bCount++
				h.lines = append(h.lines, "+"+op.text)
			}
		}
		hunks = append(hunks, h)
	}

	return hunks
}
