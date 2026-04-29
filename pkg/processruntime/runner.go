package processruntime

// Stage 描述一个可按顺序执行的启动阶段。
type Stage[S any] interface {
	Name() string
	Run(*S) error
}

// Runner 按顺序执行阶段，并在全部成功后构建 prepared 输出。
type Runner[S any, P any] struct {
	State         *S
	Stages        []Stage[S]
	BuildPrepared func(*S) P
}

// Run 按顺序执行阶段，并返回 prepared 输出、失败阶段名称和错误。
func (r Runner[S, P]) Run() (P, string, error) {
	var zero P

	state := r.State
	if state == nil {
		state = new(S)
	}

	for _, stage := range r.Stages {
		if stage == nil {
			continue
		}
		if err := stage.Run(state); err != nil {
			return zero, stage.Name(), err
		}
	}

	if r.BuildPrepared == nil {
		return zero, "", nil
	}
	return r.BuildPrepared(state), "", nil
}
