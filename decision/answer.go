package decision

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

// Answer 单个问题的答案。字段按类型选填：
// choice 看 Choice/Confidence，noul 看 Noul（P(yes)），score 看 Score/Legend。
type Answer struct {
	Type   Type   `json:"type"`
	Choice string `json:"choice,omitempty"`
	// Noul 为 P(yes)，0=否 1=是。
	Noul *float64 `json:"noul,omitempty"`
	// Score 为等级索引的概率加权期望，可落在两级之间（如 1.43）。
	Score float64 `json:"score,omitempty"`
	// Probabilities 每个选项 / 等级的概率，key 为选项名或等级索引字符串。
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	// Confidence 该答案的置信度（choice / score 类型返回）。
	Confidence float64 `json:"confidence,omitempty"`
	// Legend 等级索引 → 等级描述（score 类型返回）。
	Legend map[string]string `json:"legend,omitempty"`
}

// Label 返回选项名；服务端未回传 choice 时退化为概率最高的选项。
func (a Answer) Label() string {
	if a.Choice != "" {
		return a.Choice
	}
	name, _ := a.TopProbability()
	return name
}

// Probability 返回某个选项 / 等级的概率。
func (a Answer) Probability(key string) float64 {
	return a.Probabilities[key]
}

// TopProbability 返回概率最高的选项（或等级）及其概率。
func (a Answer) TopProbability() (string, float64) {
	var topKey string
	var topVal float64 = -1
	// 排序保证结果稳定，避免 map 遍历顺序影响概率相同时的取值。
	keys := make([]string, 0, len(a.Probabilities))
	for k := range a.Probabilities {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if v := a.Probabilities[k]; v > topVal {
			topKey, topVal = k, v
		}
	}
	if topVal < 0 {
		return "", 0
	}
	return topKey, topVal
}

// YesProbability 返回 P(yes)，非 noul 问题时为 0。
func (a Answer) YesProbability() float64 {
	if a.Noul == nil {
		return 0
	}
	return *a.Noul
}

// Yes 按 P(yes) >= 0.5 判定，阈值需要自己把握时用 YesProbability。
func (a Answer) Yes() bool {
	return a.YesProbability() >= 0.5
}

// Value 返回 score 的概率加权期望。
func (a Answer) Value() float64 {
	return a.Score
}

// Level 返回期望值四舍五入后的等级索引。
func (a Answer) Level() int {
	return int(math.Round(a.Score))
}

// LevelText 返回当前等级的描述。
func (a Answer) LevelText() string {
	return a.LevelTextAt(a.Level())
}

// LevelTextAt 返回指定索引的等级描述。
func (a Answer) LevelTextAt(i int) string {
	return a.Legend[strconv.Itoa(i)]
}

// String 便于日志输出的一行摘要。
func (a Answer) String() string {
	switch a.Type {
	case TypeNoul:
		return fmt.Sprintf("noul p_yes=%.2f", a.YesProbability())
	case TypeScore:
		return fmt.Sprintf("score=%.2f level=%d(%s) confidence=%.2f", a.Score, a.Level(), a.LevelText(), a.Confidence)
	default:
		if a.Choice == "" {
			a.Choice = a.Label()
		}
		return fmt.Sprintf("choice=%s confidence=%.2f", a.Choice, a.Confidence)
	}
}
