// Package decision 封装阿里云百炼决策模型 API（POST /compatible-mode/v1/systemone）。
// 一次前向返回分类（choice）、是非（noul）、量表（score）的判定结果与概率分布，不生成文本，
// 因此延迟与成本跟输出长度无关，适合工单分流、内容审核、Agent 路由与结果校验。
package decision

import (
	"fmt"
	"strings"
)

// Type 问题类型。
type Type string

const (
	TypeChoice Type = "choice" // 多选一
	TypeNoul   Type = "noul"   // 是否判断，返回 P(yes)
	TypeScore  Type = "score"  // 有序量表，返回等级索引的概率加权期望
)

const (
	// DefaultModel 决策模型名（preview 阶段）。
	DefaultModel = "decision-model-preview"

	// 接口限制，见文档「接口限制与建议」。
	MaxQuestions  = 16  // 单次请求问题数上限，延迟随问题数近线性增长
	MaxOptions    = 255 // choice 选项数上限
	MinScoreLevel = 2   // score 等级数下限
	MaxScoreLevel = 255 // score 等级数上限
	// MaxContext 上下文 token 上限，超长 state 由服务端拒绝（本地不做 token 估算）。
	MaxContext = 65536
)

// ChoiceOptions 选项名 → 选项描述，建议给全量选项并补 other 兜底。
type ChoiceOptions map[string]string

// ScoreLevels 从低到高的等级描述，建议 3-7 级且每级可清晰区分。
type ScoreLevels []string

// Question 单个问题定义。
type Question struct {
	Type         Type   `json:"type"`
	Instructions string `json:"instructions,omitempty"`
	// Criteria 随类型变化：choice 为 ChoiceOptions，score 为 ScoreLevels，
	// noul 可选 {"true":…,"false":…}。
	Criteria any `json:"criteria,omitempty"`
}

// NewChoice 构造多选一问题。
func NewChoice(instructions string, options ChoiceOptions) Question {
	return Question{Type: TypeChoice, Instructions: instructions, Criteria: options}
}

// NewYesNo 构造是非判断问题。
func NewYesNo(instructions string) Question {
	return Question{Type: TypeNoul, Instructions: instructions}
}

// NewYesNoWith 构造带 true/false 描述的是非判断问题。
func NewYesNoWith(instructions, trueDesc, falseDesc string) Question {
	return Question{
		Type:         TypeNoul,
		Instructions: instructions,
		Criteria:     ChoiceOptions{"true": trueDesc, "false": falseDesc},
	}
}

// NewScore 构造有序量表问题，levels 从低到高。
func NewScore(instructions string, levels ...string) Question {
	return Question{Type: TypeScore, Instructions: instructions, Criteria: ScoreLevels(levels)}
}

// Validate 校验问题与类型是否匹配。
func (q Question) Validate() error {
	switch q.Type {
	case TypeChoice:
		options, ok := q.Criteria.(ChoiceOptions)
		if !ok {
			return fmt.Errorf("decision: choice 问题的 criteria 必须是 ChoiceOptions")
		}
		if len(options) == 0 {
			return fmt.Errorf("decision: choice 问题至少需要 1 个选项")
		}
		if len(options) > MaxOptions {
			return fmt.Errorf("decision: choice 选项数 %d 超过上限 %d", len(options), MaxOptions)
		}
		for name := range options {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("decision: choice 选项名不能为空")
			}
		}
	case TypeNoul:
		if q.Criteria != nil {
			if _, ok := q.Criteria.(ChoiceOptions); !ok {
				return fmt.Errorf("decision: noul 问题的 criteria 必须是 {\"true\":…,\"false\":…}")
			}
		}
	case TypeScore:
		levels, ok := q.Criteria.(ScoreLevels)
		if !ok {
			return fmt.Errorf("decision: score 问题的 criteria 必须是 ScoreLevels")
		}
		if len(levels) < MinScoreLevel || len(levels) > MaxScoreLevel {
			return fmt.Errorf("decision: score 等级数 %d 应在 %d-%d 之间", len(levels), MinScoreLevel, MaxScoreLevel)
		}
	default:
		return fmt.Errorf("decision: 未知问题类型 %q", q.Type)
	}
	return nil
}

// Request 决策模型请求体。
type Request struct {
	Model string `json:"model"`
	// State 待决策的业务状态：工单文本、对话或结构化对象（对象会被序列化后送模型）。
	State any `json:"state"`
	// Questions 问题表，key 为调用方自定义的问题 id。
	Questions map[string]Question `json:"questions"`
}

// Validate 补齐模型名并校验请求，超长的 state 会被服务端拒绝（上下文上限 65536 token）。
func (r *Request) Validate() error {
	if strings.TrimSpace(r.Model) == "" {
		r.Model = DefaultModel
	}
	switch state := r.State.(type) {
	case nil:
		return fmt.Errorf("decision: state 不能为空")
	case string:
		if strings.TrimSpace(state) == "" {
			return fmt.Errorf("decision: state 不能为空")
		}
	}
	if len(r.Questions) == 0 {
		return fmt.Errorf("decision: questions 不能为空")
	}
	if len(r.Questions) > MaxQuestions {
		return fmt.Errorf("decision: 问题数 %d 超过上限 %d", len(r.Questions), MaxQuestions)
	}
	for id, q := range r.Questions {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("decision: 问题 id 不能为空")
		}
		if err := q.Validate(); err != nil {
			return fmt.Errorf("decision: 问题 %q: %w", id, err)
		}
	}
	return nil
}

// Usage 计量信息。
type Usage struct {
	InputTokens int `json:"input_tokens"`
}

// Response 决策模型返回体。
type Response struct {
	Model     string            `json:"model"`
	RequestID string            `json:"request_id"`
	Answers   map[string]Answer `json:"answers"`
	Usage     Usage             `json:"usage"`
	LatencyMS float64           `json:"latency_ms"`
}

// Answer 取某个问题的答案，问题不存在时返回零值。
func (r *Response) Answer(id string) Answer {
	if r == nil {
		return Answer{}
	}
	return r.Answers[id]
}

// MustAnswer 取某个问题的答案，问题不存在时报错。
func (r *Response) MustAnswer(id string) (Answer, error) {
	if r == nil {
		return Answer{}, fmt.Errorf("decision: 响应为空，缺少问题 %q", id)
	}
	a, ok := r.Answers[id]
	if !ok {
		return Answer{}, fmt.Errorf("decision: 响应缺少问题 %q", id)
	}
	return a, nil
}
