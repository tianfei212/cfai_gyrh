package workflow

import (
	"context"
	"fmt"
	"strings"

	"gyrh-go-v2/backend/internal/logger"
)

// StepContext 步骤执行上下文，用于在步骤之间传递数据
type StepContext struct {
	Ctx        context.Context    // 上下文
	Data       map[string]interface{} // 步骤间共享数据
	RollbackFuncs []func() error  // 回滚函数列表
}

// NewStepContext 创建新的步骤上下文
func NewStepContext(ctx context.Context) *StepContext {
	return &StepContext{
		Ctx:        ctx,
		Data:       make(map[string]interface{}),
		RollbackFuncs: make([]func() error, 0),
	}
}

// AddRollback 添加回滚函数
// fn: 回滚函数，按逆序执行
func (sc *StepContext) AddRollback(fn func() error) {
	sc.RollbackFuncs = append(sc.RollbackFuncs, fn)
}

// Get 获取上下文数据
func (sc *StepContext) Get(key string) (interface{}, bool) {
	v, ok := sc.Data[key]
	return v, ok
}

// Set 设置上下文数据
func (sc *StepContext) Set(key string, value interface{}) {
	sc.Data[key] = value
}

// StepError 步骤执行错误
type StepError struct {
	StepName string // 步骤名称
	Err      error  // 错误信息
}

// Error 实现 error 接口
func (e *StepError) Error() string {
	return fmt.Sprintf("步骤 [%s] 执行失败: %v", e.StepName, e.Err)
}

// Unwrap 返回底层错误
func (e *StepError) Unwrap() error {
	return e.Err
}

// StepInterface 步骤接口，定义工作流中每个步骤的行为
type StepInterface interface {
	// Execute 执行步骤逻辑
	// ctx: 步骤上下文，包含执行状态和共享数据
	// 返回: 执行结果和错误信息
	Execute(ctx *StepContext) (interface{}, error)

	// Rollback 回滚步骤执行
	// ctx: 步骤上下文
	// 返回: 回滚结果和错误信息
	Rollback(ctx *StepContext) error

	// Name 获取步骤名称
	Name() string
}

// StepBase 步骤基类，提供通用的回滚机制
type StepBase struct {
	stepName string // 步骤名称
}

// NewStepBase 创建步骤基类
func NewStepBase(name string) StepBase {
	return StepBase{stepName: name}
}

// Name 获取步骤名称
func (s *StepBase) Name() string {
	return s.stepName
}

// StepFunc 基于函数类型的步骤实现
type StepFunc struct {
	StepBase
	ExecuteFunc func(ctx *StepContext) (interface{}, error)
	RollbackFunc func(ctx *StepContext) error
}

// Execute 执行步骤
func (s *StepFunc) Execute(ctx *StepContext) (interface{}, error) {
	return s.ExecuteFunc(ctx)
}

// Rollback 执行回滚
func (s *StepFunc) Rollback(ctx *StepContext) error {
	if s.RollbackFunc != nil {
		return s.RollbackFunc(ctx)
	}
	return nil
}

// Pipeline 流水线结构，管理一系列步骤的执行
type Pipeline struct {
	name  string           // 流水线名称
	steps []StepInterface  // 步骤列表
}

// NewPipeline 创建新的流水线实例
func NewPipeline(name string) *Pipeline {
	return &Pipeline{
		name:  name,
		steps: make([]StepInterface, 0),
	}
}

// AddStep 添加步骤到流水线
// step: 要添加的步骤
// 返回: 流水线自身，支持链式调用
func (p *Pipeline) AddStep(step StepInterface) *Pipeline {
	p.steps = append(p.steps, step)
	return p
}

// AddStepFunc 添加函数类型的步骤
// name: 步骤名称
// executeFn: 执行函数
// rollbackFn: 回滚函数（可选）
// 返回: 流水线自身，支持链式调用
func (p *Pipeline) AddStepFunc(name string, executeFn func(ctx *StepContext) (interface{}, error), rollbackFn func(ctx *StepContext) error) *Pipeline {
	step := &StepFunc{
		StepBase:   NewStepBase(name),
		ExecuteFunc: executeFn,
		RollbackFunc: rollbackFn,
	}
	p.steps = append(p.steps, step)
	return p
}

// Execute 执行流水线
// ctx: 步骤上下文
// 返回: 执行结果和错误信息
func (p *Pipeline) Execute(ctx *StepContext) (interface{}, error) {
	logger.Info("开始执行流水线: %s", p.name)

	var result interface{}
	var stepErr error

	// 依次执行每个步骤
	for i, step := range p.steps {
		logger.Info("执行步骤 [%d/%d]: %s", i+1, len(p.steps), step.Name())

		stepResult, err := step.Execute(ctx)
		if err != nil {
			logger.Error("步骤 [%s] 执行失败: %v", step.Name(), err)
			stepErr = &StepError{
				StepName: step.Name(),
				Err:      err,
			}

			// 执行失败，触发回滚机制
			logger.Info("流水线执行失败，开始回滚...")
			p.rollback(ctx)
			return nil, stepErr
		}

		result = stepResult
		logger.Info("步骤 [%s] 执行成功", step.Name())
	}

	logger.Info("流水线 [%s] 执行完成", p.name)
	return result, nil
}

// rollback 执行回滚
// ctx: 步骤上下文
func (p *Pipeline) rollback(ctx *StepContext) {
	// 逆序执行已注册的回滚函数
	for i := len(ctx.RollbackFuncs) - 1; i >= 0; i-- {
		rollbackFn := ctx.RollbackFuncs[i]
		if err := rollbackFn(); err != nil {
			// 回滚失败，记录错误但继续执行其他回滚
			logger.Error("回滚函数执行失败: %v", err)
		}
	}

	// 按逆序执行每个步骤的回滚
	for i := len(p.steps) - 1; i >= 0; i-- {
		step := p.steps[i]
		logger.Info("回滚步骤: %s", step.Name())

		if err := step.Rollback(ctx); err != nil {
			// 单个步骤回滚失败不影响其他步骤回滚
			logger.Error("步骤 [%s] 回滚失败: %v", step.Name(), err)
		}
	}

	logger.Info("回滚执行完成")
}

// GetSteps 获取流水线中的所有步骤
func (p *Pipeline) GetSteps() []StepInterface {
	return p.steps
}

// String 返回流水线的字符串表示
func (p *Pipeline) String() string {
	stepNames := make([]string, len(p.steps))
	for i, step := range p.steps {
		stepNames[i] = step.Name()
	}
	return fmt.Sprintf("Pipeline[%s]: [%s]", p.name, strings.Join(stepNames, " -> "))
}