package processruntime

// ShutdownHook 是一个带名字的关闭钩子。
type ShutdownHook struct {
	name string
	run  func() error
}

// Name 返回关闭钩子的名称。
func (h ShutdownHook) Name() string {
	return h.name
}

// Run 执行关闭钩子；未配置执行函数时直接返回 nil。
func (h ShutdownHook) Run() error {
	if h.run == nil {
		return nil
	}
	return h.run()
}

// Lifecycle 收集关闭钩子，并按注册顺序执行。
type Lifecycle struct {
	hooks []ShutdownHook
}

// AddShutdownHook 注册一个关闭钩子。
func (l *Lifecycle) AddShutdownHook(name string, run func() error) {
	if l == nil || run == nil {
		return
	}
	l.hooks = append(l.hooks, ShutdownHook{name: name, run: run})
}

// Len 返回当前已注册的关闭钩子数量。
func (l Lifecycle) Len() int {
	return len(l.hooks)
}

// Run 按顺序执行所有关闭钩子。
// 单个钩子的错误会通过 onError 汇报，但不会阻止后续钩子继续执行。
func (l Lifecycle) Run(onError func(name string, err error)) {
	for _, hook := range l.hooks {
		if err := hook.Run(); err != nil && onError != nil {
			onError(hook.Name(), err)
		}
	}
}
