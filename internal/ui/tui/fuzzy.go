package tui

import (
	"sort"
	"strings"
)

// LevenshteinDistance は 2 つの文字列間のレーベンシュタイン距離を計算します
func LevenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(strings.ToLower(s1)), []rune(strings.ToLower(s2))
	len1, len2 := len(r1), len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	dp := make([][]int, len1+1)
	for i := range dp {
		dp[i] = make([]int, len2+1)
		dp[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			dp[i][j] = min(
				dp[i-1][j]+1,      // 削除
				dp[i][j-1]+1,      // 挿入
				dp[i-1][j-1]+cost, // 置換
			)
		}
	}

	return dp[len1][len2]
}

type MatchItem struct {
	Target   string
	Distance int
}

// FuzzyRank はクエリに対するレーベンシュタイン距離順（昇順）に候補をソートします
func FuzzyRank(query string, targets []string) []string {
	if query == "" {
		return targets
	}

	lowerQuery := strings.ToLower(query)
	var matches []MatchItem

	for _, target := range targets {
		dist := LevenshteinDistance(lowerQuery, target)

		// クエリが部分一致する場合は大幅にペナルティを減らして上位に持ち上げる
		if strings.Contains(strings.ToLower(target), lowerQuery) {
			dist = -100 + len(target)
		}

		matches = append(matches, MatchItem{
			Target:   target,
			Distance: dist,
		})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Distance == matches[j].Distance {
			return matches[i].Target < matches[j].Target
		}
		return matches[i].Distance < matches[j].Distance
	})

	results := make([]string, len(matches))
	for i, m := range matches {
		results[i] = m.Target
	}
	return results
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
