package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/xsigil/m4llama/internal/domain/entity"
)

type SelectedInfo struct {
	Role string
}

func ParseSelectedIndices(spec string) map[int64]SelectedInfo {
	indices := make(map[int64]SelectedInfo)
	if strings.TrimSpace(spec) == "" {
		return indices
	}

	tokens := strings.Split(spec, ",")
	for _, raw := range tokens {
		t := strings.TrimSpace(raw)
		if t == "" || t == "$" || t == "s$" || t == "u$" || t == "a$" {
			continue
		}

		role := ""
		if len(t) > 1 && (t[0] == 's' || t[0] == 'u' || t[0] == 'a') {
			switch t[0] {
			case 's':
				role = "system"
			case 'u':
				role = "user"
			case 'a':
				role = "assistant"
			}
			t = t[1:]
		}

		if strings.Contains(t, "~") {
			parts := strings.SplitN(t, "~", 2)
			start, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
			end, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
			if err1 == nil && err2 == nil && start <= end {
				for i := start; i <= end; i++ {
					indices[i] = SelectedInfo{Role: role}
				}
			}
			continue
		}

		if idx, err := strconv.ParseInt(t, 10, 64); err == nil {
			indices[idx] = SelectedInfo{Role: role}
		}
	}

	return indices
}

type TargetToken struct {
	RoleOverride string
	IsCurrent    bool
	ID           int64
	RangeStart   int64
	RangeEnd     int64
	IsRange      bool
	OrderIndex   int
}

func ParseContextSpec(spec string) ([]TargetToken, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}

	tokens := strings.Split(spec, ",")
	var targets []TargetToken

	for i, raw := range tokens {
		t := strings.TrimSpace(raw)
		if t == "" {
			continue
		}

		role := ""
		if len(t) > 1 && (t[0] == 's' || t[0] == 'u' || t[0] == 'a') {
			switch t[0] {
			case 's':
				role = "system"
			case 'u':
				role = "user"
			case 'a':
				role = "assistant"
			}
			t = t[1:]
		}

		if t == "$" {
			targets = append(targets, TargetToken{
				RoleOverride: role,
				IsCurrent:    true,
				OrderIndex:   i,
			})
			continue
		}

		if strings.Contains(t, "~") {
			parts := strings.SplitN(t, "~", 2)
			start, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
			end, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
			if err1 != nil || err2 != nil || start > end {
				return nil, fmt.Errorf("invalid range: %s", raw)
			}
			targets = append(targets, TargetToken{
				RoleOverride: role,
				IsRange:      true,
				RangeStart:   start,
				RangeEnd:     end,
				OrderIndex:   i,
			})
			continue
		}

		id, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid id: %s", t)
		}

		targets = append(targets, TargetToken{
			RoleOverride: role,
			ID:           id,
			OrderIndex:   i,
		})
	}

	return targets, nil
}

func BuildChronologicalMessages(targets []TargetToken, currentPrompt string, history []*entity.HistoryEntry) ([]entity.Message, error) {
	byID := make(map[int64]*entity.HistoryEntry)
	for _, h := range history {
		byID[h.ID] = h
	}

	type resolvedItem struct {
		entry        *entity.HistoryEntry
		roleOverride string
	}

	var historyItems []resolvedItem
	currentRole := "user"
	hasCurrent := false

	for _, target := range targets {
		if target.IsCurrent {
			hasCurrent = true
			if target.RoleOverride != "" {
				currentRole = target.RoleOverride
			}
			continue
		}

		if target.IsRange {
			for idx := target.RangeStart; idx <= target.RangeEnd; idx++ {
				h, ok := byID[idx]
				if !ok {
					continue
				}
				historyItems = append(historyItems, resolvedItem{entry: h, roleOverride: target.RoleOverride})
			}
			continue
		}

		h, ok := byID[target.ID]
		if !ok {
			return nil, fmt.Errorf("history item #%d not found", target.ID)
		}
		historyItems = append(historyItems, resolvedItem{entry: h, roleOverride: target.RoleOverride})
	}

	sort.SliceStable(historyItems, func(i, j int) bool {
		return historyItems[i].entry.ID < historyItems[j].entry.ID
	})

	var msgs []entity.Message
	for _, it := range historyItems {
		prompt := strings.TrimSpace(it.entry.ExpandedPrompt)
		comp := strings.TrimSpace(it.entry.Completion)

		if it.roleOverride != "" {
			combined := prompt
			if comp != "" {
				combined = fmt.Sprintf("%s\n\n%s", prompt, comp)
			}
			if combined != "" {
				msgs = append(msgs, entity.Message{Role: it.roleOverride, Content: combined})
			}
		} else {
			if prompt != "" {
				msgs = append(msgs, entity.Message{Role: "user", Content: prompt})
			}
			if comp != "" {
				msgs = append(msgs, entity.Message{Role: "assistant", Content: comp})
			}
		}
	}

	cleanCurrent := strings.TrimSpace(currentPrompt)
	if hasCurrent || len(targets) == 0 {
		if cleanCurrent != "" {
			msgs = append(msgs, entity.Message{Role: currentRole, Content: cleanCurrent})
		}
	} else if cleanCurrent != "" {
		msgs = append(msgs, entity.Message{Role: "user", Content: cleanCurrent})
	}

	return msgs, nil
}
