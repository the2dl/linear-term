package data

import (
	"fmt"
	"time"

	"github.com/the2dl/linear-term/internal/model"
)

var (
	projEdge       = &model.Project{Name: "Edge runtime", Color: "#9B51E0", Icon: "❖"}
	projEnrichment = &model.Project{Name: "Pipeline feeds", Color: "#27AE60", Icon: "❖"}
	projStorage    = &model.Project{Name: "Storage engine", Color: "#2F80ED", Icon: "❖"}
	projRegions    = &model.Project{Name: "Regions & failover", Color: "#EB5757", Icon: "❖"}
	projOnboarding = &model.Project{Name: "Onboarding & Setup", Color: "#F2994A", Icon: "❖"}
	projQueryLane  = &model.Project{Name: "Query planner", Color: "#F2994A", Icon: "❖"}

	lblSlow        = model.Label{Name: "slow", Color: "#9B51E0"}
	lblFast        = model.Label{Name: "fast", Color: "#EB5757"}
	lblTrain       = model.Label{Name: "train", Color: "#2F80ED"}
	lblBug         = model.Label{Name: "Bug", Color: "#EB5757"}
	lblFeature     = model.Label{Name: "Feature", Color: "#BB6BD9"}
	lblImprovement = model.Label{Name: "Improvement", Color: "#2D9CDB"}
)

// SeedIssues returns sample Delivery board issues for demo mode.
func SeedIssues() []model.Issue {
	now := time.Now()
	return []model.Issue{
		// ==================== IN PROGRESS (8) ====================
		{
			ID:          "seed-4742",
			Identifier:  "LIN-4742",
			Title:       "Single-binary airgap host — log in, ingest one event, search end-to-end",
			Description: "End-to-end validation for single-host offline deployment:\n\n- Boot single-binary bundle without external control plane\n- Authenticate local admin operator session\n- Ingest sample event via local collector socket\n- Verify immediate query visibility in search within <2s",
			Column:      model.ColInProgress,
			StateName:   "In Progress",
			Priority:    model.PriorityUrgent,
			Project:     projEdge,
			Labels:      []model.Label{lblSlow, lblFeature},
			PRs:         &model.PRInfo{Count: 4, URLs: []string{"https://github.com/acme-org/acme-core/pull/8241", "https://github.com/acme-org/acme-core/pull/8244", "https://github.com/acme-org/acme-core/pull/8251", "https://github.com/acme-org/acme-core/pull/8260"}},
			SubIssues: &model.SubIssueProgress{
				Done:  3,
				Total: 3,
				Items: []model.SubIssueItem{
					{Identifier: "LIN-4743", Title: "Bundle embedded auth verifier in standalone binary", Done: true},
					{Identifier: "LIN-4744", Title: "Wire local ingest socket to L0 storage writer", Done: true},
					{Identifier: "LIN-4745", Title: "Smoke test harness: login + 1 event + search", Done: true},
				},
			},
			Assignee:   "dan",
			CreatedStr: "Oct 27",
			UpdatedStr: "Oct 28",
			UpdatedAt:  now.Add(-25 * time.Minute),
			URL:        "https://linear.app/acme/issue/LIN-4742",
			BranchName: "dan/lin-4742-airgap-e2e",
			Comments: []model.Comment{
				{Author: "dan", Body: "All 3 sub-issues merged; running final smoke verification before moving to In Review.", CreatedAt: "10m ago"},
			},
		},
		{
			ID:          "seed-4759",
			Identifier:  "LIN-4759",
			Title:       "Portable lookup feeds: GeoIP/ASN stamping at ingest for isolated worker nodes",
			Description: "Isolated worker nodes need deterministic GeoIP/ASN metadata stamping at ingest time without calling external network services.\n\n### Requirements\n- Memory-mapped MMDB + compact FST lookup table loaded by ingest workers\n- Zero-allocation lookup path during pipeline execution\n- Hot-reload on feed bundle swap without dropping active ingest batches",
			Column:      model.ColInProgress,
			StateName:   "In Progress",
			Priority:    model.PriorityUrgent,
			Project:     projEnrichment,
			Labels:      []model.Label{lblSlow, lblBug},
			PRs:         &model.PRInfo{Count: 4, URLs: []string{"https://github.com/acme-org/acme-feeds/pull/112", "https://github.com/acme-org/acme-core/pull/8264"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-8 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4759",
			BranchName:  "dan/lin-4759-portable-lookup-feeds",
			Comments: []model.Comment{
				{Author: "agent-worker", Body: "Benchmarked mmap ASN lookup at 142ns/event on 16-core worker.", CreatedAt: "14m ago"},
				{Author: "dan", Body: "Checking archive signature verification on hot reload.", CreatedAt: "8m ago"},
			},
		},
		{
			ID:          "seed-4766",
			Identifier:  "LIN-4766",
			Title:       "Segment GC must keep pace with compaction merges at high ingest rates (disk bounded)",
			Description: "Under sustained high-throughput ingest, merged L0/L1 segments accumulate faster than the background GC sweep reclaims superseded directories, causing transient disk spikes.\n\n### Fix\n- Uncouple segment-deletion workers from compaction semaphore\n- Track superseded generation watermarks per hour partition\n- Enforce bounded disk budget with prioritized reclamation of merged L0 segments",
			Column:      model.ColInProgress,
			StateName:   "In Progress",
			Priority:    model.PriorityHigh,
			Project:     projStorage,
			Labels:      []model.Label{lblTrain, lblImprovement},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-12 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4766",
			BranchName:  "dan/lin-4766-segment-gc-pace",
		},
		{
			ID:          "seed-4765",
			Identifier:  "LIN-4765",
			Title:       "Worker ingest throughput drops from 12.5k to 7.4k eps as active partition fills",
			Description: "Profiling shows write amplification climbing after 40 minutes in the same hour partition due to repeated dictionary re-indexing across L0 flushes.\n\n### Root Cause\n- Per-flush symbol table merge scans cold segments unnecessarily\n- Fix keeps per-segment local dictionaries and defers global dictionary unification to L1 compaction",
			Column:      model.ColInProgress,
			StateName:   "In Progress",
			Priority:    model.PriorityHigh,
			Project:     projStorage,
			Labels:      []model.Label{lblTrain, lblBug},
			Assignee:    "codex",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-18 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4765",
			BranchName:  "fix/lin-4765-ingest-throughput-decay",
		},
		{
			ID:          "seed-4767",
			Identifier:  "LIN-4767",
			Title:       "Storage coordinator: CONNECTION_CLOSED aborts GC pass before lease renewal completes",
			Description: "When the object store keepalive pool recycles idle connections between GC sweeps, `CONNECTION_CLOSED` aborts the entire GC pass instead of retrying idempotent DELETE batch calls.",
			Column:      model.ColInProgress,
			StateName:   "In Progress",
			Priority:    model.PriorityNone,
			Project:     projRegions,
			Labels:      []model.Label{lblFast, lblBug},
			Assignee:    "codex",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-22 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4767",
			BranchName:  "fix/lin-4767-coordinator-gc-retry",
		},
		{
			ID:          "seed-4664",
			Identifier:  "LIN-4664",
			Title:       "Background query lane: needle substring scans 5x slower before bloom index pushdown",
			Description: "High-concurrency needle + multi-token substring queries across 24h-7d windows decode full string columns before predicate pushdown.\n\n### Optimizations\n- Bloom + trigram index skip before column page decompression\n- Dedicated low-latency background query threadpool so batch jobs never starve interactive dashboard queries",
			Column:      model.ColInProgress,
			StateName:   "In Progress",
			Priority:    model.PriorityHigh,
			Project:     projQueryLane,
			Labels:      []model.Label{lblSlow, lblImprovement},
			PRs:         &model.PRInfo{Count: 6, URLs: []string{"https://github.com/acme-org/acme-core/pull/8210", "https://github.com/acme-org/acme-core/pull/8235"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 26",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-30 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4664",
			BranchName:  "dan/lin-4664-query-lane-simd",
		},
		{
			ID:          "seed-4760",
			Identifier:  "LIN-4760",
			Title:       "Regional writer HA: enforce single-active replica and strict epoch fencing on manifest commits",
			Description: "Writer stateful deployment in regional clusters must run single-active replica with `Recreate` rollout strategy and strict epoch fencing on object store manifest commits.",
			Column:      model.ColInProgress,
			StateName:   "In Progress",
			Priority:    model.PriorityUrgent,
			Project:     projRegions,
			Labels:      []model.Label{lblFast, lblBug},
			PRs:         &model.PRInfo{Count: 1, Number: "#3194", URLs: []string{"https://github.com/acme-org/acme-infra/pull/3194"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-15 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4760",
			BranchName:  "dan/lin-4760-regional-writer-ha",
		},
		{
			ID:          "seed-4762",
			Identifier:  "LIN-4762",
			Title:       "Storage compaction: bound L1 sort-merge spill memory under bursty telemetry",
			Description: "Cap per-worker sort-merge spill buffers so simultaneous L1 compactions stay comfortably within container cgroup memory limits.",
			Column:      model.ColInProgress,
			StateName:   "In Progress",
			Priority:    model.PriorityHigh,
			Project:     projStorage,
			Labels:      []model.Label{lblTrain, lblImprovement},
			PRs:         &model.PRInfo{Count: 2, URLs: []string{"https://github.com/acme-org/acme-core/pull/8266"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-40 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4762",
			BranchName:  "dan/lin-4762-l1-spill-memory-bound",
		},

		// ==================== IN REVIEW (2) ====================
		{
			ID:          "seed-4755",
			Identifier:  "LIN-4755",
			Title:       "Worker node: ingest cgroup-OOM loop during post-compaction WAL replay",
			Description: "Uncheckpointed WAL segments accumulated during an upstream outage. Replaying all segments concurrently triggered a cgroup OOM loop on startup.\n\n### Fix\n- Stream WAL replay in bounded 64MB windows with incremental L0 flush\n- Add regression test reproducing 5GB WAL recovery under 1GB memory cap",
			Column:      model.ColInReview,
			StateName:   "In Review",
			Priority:    model.PriorityUrgent,
			Project:     projStorage,
			Labels:      []model.Label{lblFast, lblBug},
			PRs:         &model.PRInfo{Count: 2, URLs: []string{"https://github.com/acme-org/acme-core/pull/8267", "https://github.com/acme-org/acme-core/pull/8268"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-6 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4755",
			BranchName:  "dan/lin-4755-wal-replay-oom-fix",
		},
		{
			ID:          "seed-4763",
			Identifier:  "LIN-4763",
			Title:       "Fix flaky clock mock and schema fixture snapshot assertions on main branch",
			Description: "Stabilize deterministic clock mocks and schema fixture snapshots in the core integration test suite.",
			Column:      model.ColInReview,
			StateName:   "In Review",
			Priority:    model.PriorityNone,
			Labels:      []model.Label{lblFast, lblBug},
			PRs:         &model.PRInfo{Count: 1, Number: "#8269", URLs: []string{"https://github.com/acme-org/acme-core/pull/8269"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-11 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4763",
			BranchName:  "dan/lin-4763-core-test-fixes",
		},

		// ==================== ROLLING OUT (4) ====================
		{
			ID:          "seed-4749",
			Identifier:  "LIN-4749",
			Title:       "Cluster admission throughput + shard placement sizing for multi-region parity",
			Description: "Tuned cluster admission controller and shard placement sizing for throughput parity across US/EU/APAC regions.",
			Column:      model.ColRollingOut,
			StateName:   "Rolling Out",
			Priority:    model.PriorityHigh,
			Project:     projStorage,
			Labels:      []model.Label{lblTrain, lblImprovement},
			PRs:         &model.PRInfo{Count: 4, URLs: []string{"https://github.com/acme-org/acme-core/pull/8252", "https://github.com/acme-org/acme-core/pull/8254", "https://github.com/acme-org/acme-core/pull/8256", "https://github.com/acme-org/acme-core/pull/8259"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-19 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4749",
			BranchName:  "dan/lin-4749-admission-placement-sizing",
		},
		{
			ID:          "seed-4753",
			Identifier:  "LIN-4753",
			Title:       "Onboarding wizard: transform dry-run hits gateway timeout on large sample payloads",
			Description: "Move transform dry-run compilation and sample evaluation onto async worker job with streaming progress so edge proxies never hit 524 gateway timeout, and sanitize upstream HTML 5xx bodies.",
			Column:      model.ColRollingOut,
			StateName:   "Rolling Out",
			Priority:    model.PriorityUrgent,
			Project:     projOnboarding,
			Labels:      []model.Label{lblFast, lblBug},
			PRs:         &model.PRInfo{Count: 1, Number: "#8263", URLs: []string{"https://github.com/acme-org/acme-core/pull/8263"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-28 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4753",
			BranchName:  "dan/lin-4753-transform-draft-timeout",
		},
		{
			ID:          "seed-4750",
			Identifier:  "LIN-4750",
			Title:       "Partition writer: L0 schema family split + prompt merge of just-closed hour partitions",
			Description: "Split high-cardinality schema families at L0 write time and trigger immediate compaction merge as soon as an hour partition closes.",
			Column:      model.ColRollingOut,
			StateName:   "Rolling Out",
			Priority:    model.PriorityHigh,
			Project:     projStorage,
			Labels:      []model.Label{lblTrain, lblImprovement},
			PRs:         &model.PRInfo{Count: 1, Number: "#8258", URLs: []string{"https://github.com/acme-org/acme-core/pull/8258"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-35 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4750",
			BranchName:  "dan/lin-4750-schema-families-fresh-hours",
		},
		{
			ID:          "seed-4751",
			Identifier:  "LIN-4751",
			Title:       "Query engine: fold bounded distinct-count into primary scan + enforce per-query thread budget",
			Description: "Fold bounded HyperLogLog/distinct-count `dc()` aggregations directly into the primary scan pipeline and enforce per-query thread budgets.",
			Column:      model.ColRollingOut,
			StateName:   "Rolling Out",
			Priority:    model.PriorityHigh,
			Project:     projStorage,
			Labels:      []model.Label{lblTrain, lblImprovement},
			PRs:         &model.PRInfo{Count: 2, URLs: []string{"https://github.com/acme-org/acme-core/pull/8255", "https://github.com/acme-org/acme-core/pull/8261"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-42 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4751",
			BranchName:  "dan/lin-4751-fold-bounded-dc",
		},

		// ==================== DONE (4 Past week) ====================
		{
			ID:          "seed-4758",
			Identifier:  "LIN-4758",
			Title:       "Coordinator: startup leader check is single-attempt — restarts on transient lease contention",
			Description: "Added exponential backoff with jitter on startup leader lease acquisition before exiting pod.",
			Column:      model.ColDone,
			StateName:   "Done",
			Priority:    model.PriorityUrgent,
			Project:     projRegions,
			Labels:      []model.Label{lblFast, lblBug},
			PRs:         &model.PRInfo{Count: 1, Number: "#3192", URLs: []string{"https://github.com/acme-org/acme-infra/pull/3192"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-50 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4758",
			BranchName:  "dan/lin-4758-leader-check-backoff",
		},
		{
			ID:          "seed-4761",
			Identifier:  "LIN-4761",
			Title:       "Region release controller refuses convergence HA paths as non-fast-forward",
			Description: "Fixed release convergence verifier to accept regional HA manifest promotions.",
			Column:      model.ColDone,
			StateName:   "Done",
			Priority:    model.PriorityUrgent,
			Project:     projRegions,
			Labels:      []model.Label{lblFast, lblBug},
			PRs:         &model.PRInfo{Count: 1, Number: "#3193", URLs: []string{"https://github.com/acme-org/acme-infra/pull/3193"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-55 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4761",
			BranchName:  "dan/lin-4761-release-control-convergence",
		},
		{
			ID:          "seed-4748",
			Identifier:  "LIN-4748",
			Title:       "Control plane attestation rejected during rolling schema migration window",
			Description: "Updated control plane attestation verifier to accept adjacent schema digest versions during rolling regional upgrades.",
			Column:      model.ColDone,
			StateName:   "Done",
			Priority:    model.PriorityUrgent,
			Project:     projRegions,
			Labels:      []model.Label{lblFast, lblBug},
			PRs:         &model.PRInfo{Count: 2, URLs: []string{"https://github.com/acme-org/acme-infra/pull/3189", "https://github.com/acme-org/acme-infra/pull/3190"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-70 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4748",
			BranchName:  "dan/lin-4748-control-attestation",
		},
		{
			ID:          "seed-4754",
			Identifier:  "LIN-4754",
			Title:       "Onboarding wizard: \"Acknowledge warnings to deploy\" blocks activation when warning count is 0",
			Description: "Fixed state machine guard in Onboarding Wizard where an empty warning slice after re-validation left the acknowledge checkbox gate locked.",
			Column:      model.ColDone,
			StateName:   "Done",
			Priority:    model.PriorityUrgent,
			Project:     projOnboarding,
			Labels:      []model.Label{lblFast, lblBug},
			PRs:         &model.PRInfo{Count: 1, Number: "#8250", URLs: []string{"https://github.com/acme-org/acme-core/pull/8250"}},
			Assignee:    "dan",
			CreatedStr:  "Oct 28",
			UpdatedStr:  "Oct 28",
			UpdatedAt:   now.Add(-85 * time.Minute),
			URL:         "https://linear.app/acme/issue/LIN-4754",
			BranchName:  "dan/lin-4754-wizard-ack-warnings",
		},
	}
}

// SimulateLiveEvent mutates one issue in demo mode so the user can see live card transitions and flashes.
func SimulateLiveEvent(issues []model.Issue, step int) ([]model.Issue, string) {
	if len(issues) == 0 {
		return issues, ""
	}
	now := time.Now()
	idx := step % len(issues)
	target := &issues[idx]
	target.UpdatedAt = now
	target.UpdatedStr = "Just now"
	target.FlashUntil = now.Add(7 * time.Second)

	switch step % 3 {
	case 0:
		// Bump PR count or add a comment
		if target.PRs == nil {
			target.PRs = &model.PRInfo{Count: 1, Number: fmt.Sprintf("#%d", 8270+(step%20))}
		}
		target.FlashTag = "PR SYNC"
		target.Comments = append(target.Comments, model.Comment{
			Author:    "linear-bot",
			Body:      fmt.Sprintf("CI checks passed on branch `%s`.", target.BranchName),
			CreatedAt: "Just now",
		})
		return issues, fmt.Sprintf("⚡ Live sync: %s updated (CI / PR check)", target.Identifier)
	case 1:
		// Move an issue one step forward in workflow and flash it
		nextCol := target.Column
		switch target.Column {
		case model.ColInProgress:
			nextCol = model.ColInReview
		case model.ColInReview:
			nextCol = model.ColRollingOut
		case model.ColRollingOut:
			nextCol = model.ColDone
		case model.ColDone:
			nextCol = model.ColInProgress
		}
		oldCol := target.Column
		target.Column = nextCol
		target.StateName = string(nextCol)
		target.FlashTag = "MOVED"
		return issues, fmt.Sprintf("⚡ Live transition: %s moved %s → %s", target.Identifier, oldCol, nextCol)
	default:
		target.FlashTag = "UPDATED"
		return issues, fmt.Sprintf("⚡ Live update: %s refreshed", target.Identifier)
	}
}
