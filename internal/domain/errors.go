package domain

import "errors"

var (
	ErrNotFound          = errors.New("资源不存在")
	ErrConflict          = errors.New("修订冲突")
	ErrInvalidState      = errors.New("当前状态不允许该操作")
	ErrValidation        = errors.New("输入校验失败")
	ErrSealed            = errors.New("批次已封存，只允许读取")
	ErrDuplicateSequence = errors.New("观测序号重复")
	ErrTimeRegression    = errors.New("采集时间早于已有观测")
	ErrDutySeparation    = errors.New("复核员参与过批次操作，不满足职责分离")
	ErrIdempotency       = errors.New("request_id 已用于不同请求")
)

type RuleError struct {
	Code    string
	Message string
}

func (e *RuleError) Error() string { return e.Message }

func NewRuleError(code, message string) error {
	return &RuleError{Code: code, Message: message}
}
