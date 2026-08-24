package model

import "errors"

// 业务错误定义：HTTP 层据此映射为对应状态码。
var (
	// ErrNotFound 目标实体不存在。
	ErrNotFound = errors.New("not found")
	// ErrConflict 状态机流转或唯一性约束冲突。
	ErrConflict = errors.New("conflict")
	// ErrInvalid 输入非法（坐标越界、时间倒退、震级越界等）。
	ErrInvalid = errors.New("invalid argument")
	// ErrArchived 对已封存实体进行写操作。
	ErrArchived = errors.New("entity archived, read-only")
	// ErrDuplicate 幂等键冲突（同目录重复事件提交）。
	ErrDuplicate = errors.New("duplicate key")
	// ErrUnstable 定位不稳事件被要求参与高置信判定。
	ErrUnstable = errors.New("event location unstable")
)

// WrapError 把底层错误归类为业务错误；已有业务错误原样返回。
func WrapError(err error) error {
	if err == nil {
		return nil
	}
	for _, be := range []error{
		ErrNotFound, ErrConflict, ErrInvalid, ErrArchived, ErrDuplicate, ErrUnstable,
	} {
		if errors.Is(err, be) {
			return err
		}
	}
	return err
}
