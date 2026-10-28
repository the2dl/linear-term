package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/the2dl/linear-term/internal/model"
)

const graphqlEndpoint = "https://api.linear.app/graphql"

var prNumRegex = regexp.MustCompile(`/pull/(\d+)`)

// Client handles Linear GraphQL queries and mutations.
type Client struct {
	Token      string
	TeamKey    string
	HTTPClient *http.Client
}

// ResolveToken finds a Linear API token from CLI flag, environment variables, or ~/.config/linear-term/token.
func ResolveToken(explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit)
	}
	for _, envName := range []string{"LINEAR_API_KEY", "LINEAR_TOKEN", "LINEAR_API_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(envName)); v != "" {
			return v
		}
	}
	if cfgDir, err := os.UserConfigDir(); err == nil {
		b, err := os.ReadFile(filepath.Join(cfgDir, "linear-term", "token"))
		if err == nil && strings.TrimSpace(string(b)) != "" {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

// SaveToken persists the Linear API token to ~/.config/linear-term/token with 0600 permissions.
func SaveToken(token string) error {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(cfgDir, "linear-term")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "token"), []byte(strings.TrimSpace(token)+"\n"), 0o600)
}

func NewClient(token, teamKey string) *Client {
	return &Client{
		Token:   strings.TrimSpace(token),
		TeamKey: strings.TrimSpace(teamKey),
		HTTPClient: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlError struct {
	Message string `json:"message"`
}

func (c *Client) do(ctx context.Context, query string, vars map[string]any, out any) error {
	bodyBytes, err := json.Marshal(gqlRequest{Query: query, Variables: vars})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("linear API HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []gqlError      `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("linear GraphQL error: %s", envelope.Errors[0].Message)
	}
	if out != nil && len(envelope.Data) > 0 {
		return json.Unmarshal(envelope.Data, out)
	}
	return nil
}

// BoardResponse contains fetched workspace info, workflow state map, and issues.
type BoardResponse struct {
	WorkspaceName string
	Columns       []model.ColumnMeta
	Issues        []model.Issue
}

const boardQuery = `
query FetchDeliveryBoard($first: Int!) {
  organization {
    name
  }
  workflowStates(first: 100) {
    nodes {
      id
      name
      type
      position
    }
  }
  issues(
    first: $first
    orderBy: updatedAt
    filter: {
      state: { name: { nin: ["Canceled", "Cancelled"] } }
      labels: { name: { in: ["fast", "slow", "train"] } }
    }
  ) {
    nodes {
      id
      identifier
      title
      description
      priority
      url
      branchName
      createdAt
      updatedAt
      completedAt
      state {
        id
        name
        type
      }
      assignee {
        name
        displayName
      }
      project {
        name
        color
      }
      labels {
        nodes {
          name
          color
          parent {
            name
          }
        }
      }
      attachments {
        nodes {
          title
          url
          subtitle
        }
      }
      children {
        nodes {
          identifier
          title
          state {
            type
          }
        }
      }
      comments(first: 10) {
        nodes {
          body
          createdAt
          user {
            displayName
            name
          }
        }
      }
    }
  }
}
`

// FetchBoard retrieves live issues and workflow states from Linear.
func (c *Client) FetchBoard(ctx context.Context) (*BoardResponse, error) {
	var data struct {
		Organization struct {
			Name string `json:"name"`
		} `json:"organization"`
		WorkflowStates struct {
			Nodes []struct {
				ID       string  `json:"id"`
				Name     string  `json:"name"`
				Type     string  `json:"type"`
				Position float64 `json:"position"`
			} `json:"nodes"`
		} `json:"workflowStates"`
		Issues struct {
			Nodes []struct {
				ID          string `json:"id"`
				Identifier  string `json:"identifier"`
				Title       string `json:"title"`
				Description string `json:"description"`
				Priority    int    `json:"priority"`
				URL         string `json:"url"`
				BranchName  string `json:"branchName"`
				CreatedAt   string `json:"createdAt"`
				UpdatedAt   string `json:"updatedAt"`
				State       struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Type string `json:"type"`
				} `json:"state"`
				Assignee *struct {
					Name        string `json:"name"`
					DisplayName string `json:"displayName"`
				} `json:"assignee"`
				Project *struct {
					Name  string `json:"name"`
					Color string `json:"color"`
				} `json:"project"`
				Labels struct {
					Nodes []struct {
						Name  string `json:"name"`
						Color string `json:"color"`
					} `json:"nodes"`
				} `json:"labels"`
				Attachments struct {
					Nodes []struct {
						Title    string `json:"title"`
						URL      string `json:"url"`
						Subtitle string `json:"subtitle"`
					} `json:"nodes"`
				} `json:"attachments"`
				Children struct {
					Nodes []struct {
						Identifier string `json:"identifier"`
						Title      string `json:"title"`
						State      struct {
							Type string `json:"type"`
						} `json:"state"`
					} `json:"nodes"`
				} `json:"children"`
				Comments struct {
					Nodes []struct {
						Body      string `json:"body"`
						CreatedAt string `json:"createdAt"`
						User      *struct {
							DisplayName string `json:"displayName"`
							Name        string `json:"name"`
						} `json:"user"`
					} `json:"nodes"`
				} `json:"comments"`
			} `json:"nodes"`
		} `json:"issues"`
	}

	if err := c.do(ctx, boardQuery, map[string]any{"first": 100}, &data); err != nil {
		return nil, err
	}

	cols := model.DefaultColumns()
	for i := range cols {
		for _, st := range data.WorkflowStates.Nodes {
			if MapStateToColumn(st.Name, st.Type) == cols[i].Kind && cols[i].StateID == "" {
				cols[i].StateID = st.ID
			}
		}
	}

	oneWeekAgo := time.Now().Add(-7 * 24 * time.Hour)
	var issues []model.Issue

	for _, n := range data.Issues.Nodes {
		if c.TeamKey != "" && !strings.HasPrefix(strings.ToUpper(n.Identifier), strings.ToUpper(c.TeamKey)+"-") {
			continue
		}
		// Status is not Cancelled
		stateLower := strings.ToLower(strings.TrimSpace(n.State.Name))
		if stateLower == "canceled" || stateLower == "cancelled" {
			continue
		}
		col := MapStateToColumn(n.State.Name, n.State.Type)
		createdTime, _ := time.Parse(time.RFC3339, n.CreatedAt)
		updatedTime, _ := time.Parse(time.RFC3339, n.UpdatedAt)

		// Match screenshot behavior: Done and Duplicate columns show "Past week"
		if (col == model.ColDone || col == model.ColDuplicate) && !updatedTime.IsZero() && updatedTime.Before(oneWeekAgo) {
			continue
		}

		iss := model.Issue{
			ID:          n.ID,
			Identifier:  n.Identifier,
			Title:       n.Title,
			Description: n.Description,
			Column:      col,
			StateID:     n.State.ID,
			StateName:   n.State.Name,
			Priority:    model.Priority(n.Priority),
			CreatedStr:  formatShortDate(createdTime),
			UpdatedStr:  formatShortDate(updatedTime),
			UpdatedAt:   updatedTime,
			URL:         n.URL,
			BranchName:  n.BranchName,
		}

		if n.Assignee != nil {
			name := n.Assignee.DisplayName
			if name == "" {
				name = n.Assignee.Name
			}
			iss.Assignee = shortAssignee(name)
		}

		if n.Project != nil && n.Project.Name != "" {
			color := n.Project.Color
			if color == "" {
				color = "#2F80ED"
			}
			iss.Project = &model.Project{
				Name:  n.Project.Name,
				Color: color,
				Icon:  "❖",
			}
		}

		for _, l := range n.Labels.Nodes {
			iss.Labels = append(iss.Labels, model.Label{
				Name:  l.Name,
				Color: l.Color,
			})
		}

		// Extract linked GitHub PRs from attachments
		var prURLs, prTitles []string
		var lastPRNum string
		for _, a := range n.Attachments.Nodes {
			if strings.Contains(a.URL, "/pull/") || strings.Contains(strings.ToLower(a.Title), "pull request") || strings.Contains(strings.ToLower(a.Subtitle), "pr") {
				prURLs = append(prURLs, a.URL)
				prTitles = append(prTitles, a.Title)
				if m := prNumRegex.FindStringSubmatch(a.URL); len(m) > 1 {
					lastPRNum = "#" + m[1]
				}
			}
		}
		if len(prURLs) > 0 {
			prInfo := &model.PRInfo{
				Count:  len(prURLs),
				URLs:   prURLs,
				Titles: prTitles,
			}
			if len(prURLs) == 1 && lastPRNum != "" {
				prInfo.Number = lastPRNum
			}
			iss.PRs = prInfo
		}

		// Sub-issues
		if len(n.Children.Nodes) > 0 {
			sub := &model.SubIssueProgress{
				Total: len(n.Children.Nodes),
			}
			for _, ch := range n.Children.Nodes {
				done := ch.State.Type == "completed" || ch.State.Type == "canceled"
				if done {
					sub.Done++
				}
				sub.Items = append(sub.Items, model.SubIssueItem{
					Identifier: ch.Identifier,
					Title:      ch.Title,
					Done:       done,
				})
			}
			iss.SubIssues = sub
		}

		for _, cm := range n.Comments.Nodes {
			author := "user"
			if cm.User != nil {
				if cm.User.DisplayName != "" {
					author = cm.User.DisplayName
				} else if cm.User.Name != "" {
					author = cm.User.Name
				}
			}
			t, _ := time.Parse(time.RFC3339, cm.CreatedAt)
			iss.Comments = append(iss.Comments, model.Comment{
				Author:    author,
				Body:      cm.Body,
				CreatedAt: formatShortDate(t),
			})
		}

		issues = append(issues, iss)
	}

	wsName := data.Organization.Name
	if wsName == "" {
		wsName = "Acme"
	}

	return &BoardResponse{
		WorkspaceName: wsName,
		Columns:       cols,
		Issues:        issues,
	}, nil
}

// UpdateIssueState updates an issue's workflow state in Linear.
func (c *Client) UpdateIssueState(ctx context.Context, issueID, stateID string) error {
	if stateID == "" {
		return fmt.Errorf("missing target workflow state ID")
	}
	const mut = `
mutation UpdateIssueState($id: String!, $stateId: String!) {
  issueUpdate(id: $id, input: { stateId: $stateId }) {
    success
  }
}
`
	return c.do(ctx, mut, map[string]any{"id": issueID, "stateId": stateID}, nil)
}

// UpdateIssuePriority updates an issue's priority in Linear.
func (c *Client) UpdateIssuePriority(ctx context.Context, issueID string, priority model.Priority) error {
	const mut = `
mutation UpdateIssuePriority($id: String!, $priority: Int!) {
  issueUpdate(id: $id, input: { priority: $priority }) {
    success
  }
}
`
	return c.do(ctx, mut, map[string]any{"id": issueID, "priority": int(priority)}, nil)
}

// MapStateToColumn maps a Linear state name & type to our 7 canonical Delivery board columns.
func MapStateToColumn(stateName, stateType string) model.ColumnKind {
	lower := strings.ToLower(strings.TrimSpace(stateName))
	switch {
	case strings.Contains(lower, "review"):
		return model.ColInReview
	case strings.Contains(lower, "rolling") || strings.Contains(lower, "rollout") || strings.Contains(lower, "deploy"):
		return model.ColRollingOut
	case strings.Contains(lower, "progress") || strings.Contains(lower, "started"):
		return model.ColInProgress
	case strings.Contains(lower, "todo") || strings.Contains(lower, "to do") || strings.Contains(lower, "ready"):
		return model.ColTodo
	case strings.Contains(lower, "backlog") || strings.Contains(lower, "triage"):
		return model.ColBacklog
	case strings.Contains(lower, "duplicate") || strings.Contains(lower, "cancel"):
		return model.ColDuplicate
	case strings.Contains(lower, "done") || strings.Contains(lower, "complete") || strings.Contains(lower, "shipped"):
		return model.ColDone
	}

	switch strings.ToLower(stateType) {
	case "backlog", "triage":
		return model.ColBacklog
	case "unstarted":
		return model.ColTodo
	case "started":
		return model.ColInProgress
	case "completed":
		return model.ColDone
	case "canceled":
		return model.ColDuplicate
	default:
		return model.ColInProgress
	}
}

func formatShortDate(t time.Time) string {
	if t.IsZero() {
		return "Oct 28"
	}
	return t.Format("Jan 02")
}

func shortAssignee(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if idx := strings.IndexAny(name, "._ "); idx > 0 {
		return name[:idx]
	}
	return name
}
