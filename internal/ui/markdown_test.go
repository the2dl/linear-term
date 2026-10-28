package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/the2dl/linear-term/internal/model"
)

func TestRenderMarkdownScreenshotCase(t *testing.T) {
	desc := `## Problem

In the 1 h [LIN-4689](https://linear.app/acme/issue/LIN-4689/storage-1-tbday-soak-24-h-sustained-ingest) v3 soak at 12.5k eps (evidence: ` + "`~/.cache/bench-work/lin4689/run-v3/results/sustain/`" + `):

* **Disk grew fast.** The cell volume grew from 12.7 to 61.2 GiB, and ingest_parts from 4.4 to 38.7 GiB (to 36.7 GiB in the last 15 min). Prevalence was \~9.5 GiB.
* **GC deleted almost nothing.** Sanctioned segment GC deleted only 195 part files in two passes (12:51 and 13:12 UTC), while 131 compaction merges committed.

## Audit (storage-worker GC + supersede paths)

* GC pass cadence: fixed interval, or driven by the supersede rate?
* Whether local ingest_parts (pre-upload L0 on local disk) is cleaned only after upload or only after merge.`

	comment := `**v3 evidence**
- Run data: ` + "`~/.cache/bench-work/lin4689/run-v3/results/sustain/`" + `, which holds ` + "`storage-cell.log`" + ` and ` + "`metrics.jsonl`" + ` (` + "`part_gc_deleted_total`" + ` over time).
- Synced to [GitHub issue](https://github.com/acme-org/acme-core/issues/8267).`

	iss := model.Issue{
		Identifier:  "LIN-4766",
		Title:       "Segment GC must keep pace with merges at high eps (disk bounded)",
		Description: desc,
		Column:      model.ColInProgress,
		Priority:    model.PriorityHigh,
		CreatedStr:  "Oct 28",
		UpdatedStr:  "Oct 28",
		Comments: []model.Comment{
			{Author: "dan", Body: comment, CreatedAt: "Oct 28"},
		},
	}

	rendered := RenderDetailModal(iss, 100, 60, 0)
	plain := ansi.Strip(rendered)

	// Raw markdown tokens should be transformed, not left literal
	for _, forbidden := range []string{
		"## Problem",
		"## Audit",
		"**Disk grew fast.**",
		"**v3 evidence**",
		`\~9.5`,
		"https://linear.app/acme/issue/LIN-4689",
		"`storage-cell.log`",
	} {
		if strings.Contains(plain, forbidden) {
			t.Fatalf("expected markdown syntax %q to be formatted away, got:\n%s", forbidden, plain)
		}
	}

	// Formatted elements should be present
	for _, want := range []string{
		"▸ Problem",
		"▸ Audit (storage-worker GC + supersede paths)",
		"LIN-4689",
		"• Disk grew fast.",
		"~9.5 GiB",
		"ingest_parts",
		"part_gc_deleted_total",
		"v3 evidence",
		"GitHub issue (#8267)",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("expected rendered modal to contain %q, got:\n%s", want, plain)
		}
	}
}
