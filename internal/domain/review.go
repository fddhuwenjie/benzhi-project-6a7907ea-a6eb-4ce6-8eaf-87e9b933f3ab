package domain

import (
	"strings"
	"time"
)

type QualificationReview struct {
	ReviewID           string            `json:"review_id"`
	BatchID            string            `json:"batch_id"`
	ReviewerID         string            `json:"reviewer_id"`
	Decision           ReviewDecision    `json:"decision"`
	ChecklistResults   []ChecklistResult `json:"checklist_results"`
	Rationale          string            `json:"rationale"`
	EvidenceRootDigest string            `json:"evidence_root_digest"`
	DecidedAt          time.Time         `json:"decided_at"`
}

func (r QualificationReview) Validate() error {
	if strings.TrimSpace(r.ReviewID) == "" || strings.TrimSpace(r.BatchID) == "" || strings.TrimSpace(r.ReviewerID) == "" {
		return NewRuleError("invalid_review_identity", "复核标识、批次标识和复核员不能为空")
	}
	if r.Decision != DecisionApprove && r.Decision != DecisionReject {
		return NewRuleError("invalid_review_decision", "复核结论必须为 approve 或 reject")
	}
	if len(r.ChecklistResults) == 0 || strings.TrimSpace(r.Rationale) == "" {
		return NewRuleError("incomplete_review", "复核清单和结论依据不能为空")
	}
	for _, item := range r.ChecklistResults {
		if strings.TrimSpace(item.Item) == "" {
			return NewRuleError("invalid_checklist", "检查项名称不能为空")
		}
		if r.Decision == DecisionApprove && !item.Passed {
			return NewRuleError("failed_approval_check", "批准结论要求全部检查项通过")
		}
	}
	if r.DecidedAt.IsZero() {
		return NewRuleError("invalid_decision_time", "裁定时间不能为空")
	}
	return nil
}
