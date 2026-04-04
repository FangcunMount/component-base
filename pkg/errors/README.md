# Errors Package

一个功能全面的Go错误处理包，基于 `github.com/pkg/errors` 扩展而来，增加了错误码支持、聚合错误处理等企业级特性。

## 🚀 功能特性

- ✅ **基础错误处理** - 创建、包装、堆栈跟踪
- ✅ **错误码系统** - HTTP状态码映射、用户友好信息
- ✅ **聚合错误处理** - 批量错误收集和处理
- ✅ **堆栈跟踪** - 详细的调用栈信息
- ✅ **Go 1.13+ 兼容** - 支持新的错误处理方式
- ✅ **格式化输出** - 多种错误显示格式

## 📦 包结构

```text
pkg/errors/
├── errors.go      # 基础错误处理 (365行)
├── code.go        # 错误码系统 (139行)
├── aggregate.go   # 聚合错误处理 (235行)
├── stack.go       # 堆栈跟踪 (177行)
├── sets.go        # 字符串集合工具 (195行)
├── format.go      # 格式化功能 (204行)
└── go113.go       # Go 1.13兼容性 (38行)
```

## 🔧 核心API

### 1. 基础错误处理

```go
// 创建新错误
err := errors.New("something went wrong")
err := errors.Errorf("failed to process %s", filename)

// 包装错误，添加上下文和堆栈跟踪
err := errors.Wrap(originalErr, "failed to read file")
err := errors.Wrapf(originalErr, "failed to process file %s", filename)

// 添加堆栈跟踪或消息（不添加堆栈）
err := errors.WithStack(originalErr)
err := errors.WithMessage(originalErr, "additional context")

// 获取根本原因
rootErr := errors.Cause(err)
```

### 2. 错误码系统

错误码系统提供了业务错误与 HTTP 状态码的映射机制：

- **已知业务错误**：有明确错误码的业务错误，默认返回 HTTP 200，错误信息通过响应体中的错误码和消息传递
- **未知系统错误**：没有错误码的系统级错误（如 panic、网络故障等），返回 HTTP 500

这种设计可以让客户端更容易区分业务错误和系统错误：

- HTTP 200 + 错误码：业务层面的可预期错误，客户端可以根据错误码做对应处理
- HTTP 500：系统级严重错误，客户端应该进行重试或报警

```go
// 定义错误码
const (
    CodeUserNotFound = 10001
    CodeInvalidParam = 10002
)

// 注册错误码（不指定 HTTP 状态码，默认返回 200）
type UserNotFoundCoder struct{}
func (c UserNotFoundCoder) Code() int { return CodeUserNotFound }
func (c UserNotFoundCoder) HTTPStatus() int { return 0 } // 返回 0 表示使用默认值 200
func (c UserNotFoundCoder) String() string { return "用户不存在" }
func (c UserNotFoundCoder) Reference() string { return "https://docs.example.com/errors#10001" }

// 如果业务需要，也可以显式指定其他 HTTP 状态码
type InvalidParamCoder struct{}
func (c InvalidParamCoder) Code() int { return CodeInvalidParam }
func (c InvalidParamCoder) HTTPStatus() int { return http.StatusBadRequest } // 显式指定 400
func (c InvalidParamCoder) String() string { return "参数无效" }
func (c InvalidParamCoder) Reference() string { return "https://docs.example.com/errors#10002" }

errors.Register(UserNotFoundCoder{})
errors.Register(InvalidParamCoder{})

// 使用错误码
err := errors.WithCode(CodeUserNotFound, "用户ID: %d", userID)
err := errors.WrapC(originalErr, CodeInvalidParam, "参数验证失败")

// 检查错误码
if errors.IsCode(err, CodeUserNotFound) {
    // 处理用户不存在的情况
}

// 解析错误码信息
coder := errors.ParseCoder(err)
if coder != nil {
    log.Printf("错误码: %d, HTTP状态: %d, 消息: %s", 
               coder.Code(), coder.HTTPStatus(), coder.String())
}
```

适配于 `github.com/FangcunMount/component-base/errors` 错误包的错误码实现。

#### Code 设计规范

Code 代码从 100101 开始，1000 以下为 `github.com/FangcunMount/component-base/errors` 保留 code.

错误代码说明：100101

```text
+ 10: 服务
+ 01: 模块
+ 01: 模块下的错误码序号，每个模块可以注册 100 个错误
```

##### 服务和模块说明

|服务|模块|说明|
|----|----|----|
|10|00|通用 - 基本错误|
|10|01|通用 - 数据库类错误|
|10|02|通用 - 认证授权类错误|
|10|03|通用 - 加解码类错误|
|11|00|iam-apiserver服务 - 用户相关(模块)错误|
|11|01|iam-apiserver服务 - 密钥相关(模块)错误|

> **通用** - 所有服务都适用的错误，提高复用性，避免重复造轮子

#### 错误描述规范

错误描述包括：对外的错误描述和对内的错误描述两部分。

##### 对外的错误描述

- 对外暴露的错误，统一大写开头，结尾不要加`.`
- 对外暴露的错误，要简洁，并能准确说明问题
- 对外暴露的错误说明，应该是 `该怎么做` 而不是 `哪里错了`

##### 对内的错误描述

- 告诉用户他们可以做什么，而不是告诉他们不能做什么。
- 当声明一个需求时，用 must 而不是 should。例如，must be greater than 0、must match regex '[a-z]+'。
- 当声明一个格式不对时，用 must not。例如，must not contain。
- 当声明一个动作时用 may not。例如，may not be specified when otherField is empty、only name may be specified。
- 引用文字字符串值时，请在单引号中指示文字。例如，ust not contain '..'。
- 当引用另一个字段名称时，请在反引号中指定该名称。例如，must be greater than request。
- 指定不等时，请使用单词而不是符号。例如，must be less than 256、must be greater than or equal to 0 (不要用 larger than、bigger than、more than、higher than)。
- 指定数字范围时，请尽可能使用包含范围。
- 建议 Go 1.13 以上，error 生成方式为 fmt.Errorf("module xxx: %w", err)。
- 错误描述用小写字母开头，结尾不要加标点符号。

> 错误信息是直接暴露给用户的，不能包含敏感信息

#### 错误记录规范

在错误产生的最原始位置调用日志，打印错误信息，其它位置直接返回。

当错误发生时，调用log包打印错误，通过log包的caller功能，可以定位到log语句的位置，也即能够定位到错误发生的位置。当使用这种方式来打印日志时，需要中遵循以下规范：

- 只在错误产生的最初位置打印日志，其它地方直接返回错误，不需要再对错误进行封装。
- 当代码调用第三方包的函数时，第三方包函数出错时，打印错误信息。比如：

```go
if err := os.Chdir("/root"); err != nil {
    log.Errorf("change dir failed: %v", err)
}
```

### 3. 聚合错误处理

```go
// 创建聚合错误
var errs []error
for _, item := range items {
    if err := processItem(item); err != nil {
        errs = append(errs, err)
    }
}
if len(errs) > 0 {
    return errors.NewAggregate(errs)
}

// 并发错误收集
err := errors.AggregateGoroutines(
    func() error { return task1() },
    func() error { return task2() },
    func() error { return task3() },
)

// 错误过滤
filtered := errors.FilterOut(err, func(err error) bool {
    return err == io.EOF  // 过滤掉 EOF 错误
})

// 检查聚合错误
if agg, ok := err.(errors.Aggregate); ok {
    for _, e := range agg.Errors() {
        log.Error(e)
    }
}
```

### 4. 堆栈跟踪

```go
// 格式化选项
fmt.Printf("%s\n", err)     // 基本错误信息
fmt.Printf("%v\n", err)     // 同 %s
fmt.Printf("%+v\n", err)    // 详细信息（包含堆栈跟踪）

// 获取堆栈跟踪
type stackTracer interface {
    StackTrace() errors.StackTrace
}

if err, ok := err.(stackTracer); ok {
    for _, f := range err.StackTrace() {
        fmt.Printf("%+s:%d\n", f, f)
    }
}
```

### 5. Go 1.13+ 兼容性

```go
// 错误检查
if errors.Is(err, io.EOF) {
    // 处理 EOF
}

// 错误断言
var pathErr *os.PathError
if errors.As(err, &pathErr) {
    // 处理路径错误
}

// 错误展开
underlying := errors.Unwrap(err)
```

## 💡 使用场景

### RESTful API 错误处理

在 RESTful API 中，正确处理业务错误和系统错误：

```go
// 定义错误码
const (
    ErrCodeUserNotFound = 100001
    ErrCodeInvalidParam = 100002
)

// 注册错误码
type UserNotFoundCoder struct{}
func (c UserNotFoundCoder) Code() int { return ErrCodeUserNotFound }
func (c UserNotFoundCoder) HTTPStatus() int { return 0 } // 0 表示使用默认 200
func (c UserNotFoundCoder) String() string { return "用户不存在" }
func (c UserNotFoundCoder) Reference() string { return "https://docs.example.com/errors" }

func init() {
    errors.Register(UserNotFoundCoder{})
}

// 业务逻辑
func GetUser(id int) (*User, error) {
    if id <= 0 {
        return nil, errors.WithCode(ErrCodeInvalidParam, "invalid user id: %d", id)
    }
    
    user, err := userRepo.GetByID(id)
    if err == sql.ErrNoRows {
        return nil, errors.WithCode(ErrCodeUserNotFound, "user not found: %d", id)
    }
    if err != nil {
        // 数据库错误，没有错误码，会返回 500
        return nil, errors.Wrap(err, "failed to get user from database")
    }
    
    return user, nil
}

// HTTP 处理器
func HandleGetUser(w http.ResponseWriter, r *http.Request) {
    id := extractID(r)
    user, err := GetUser(id)
    
    if err != nil {
        coder := errors.ParseCoder(err)
        statusCode := coder.HTTPStatus() // 业务错误返回 200，系统错误返回 500
        
        response := map[string]interface{}{
            "code":    coder.Code(),
            "message": coder.String(),
        }
        
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(statusCode)
        json.NewEncoder(w).Encode(response)
        return
    }
    
    // 成功响应
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "code": 0,
        "data": user,
    })
}
```

**响应示例：**

```json
// 业务错误：HTTP 200
{
  "code": 100001,
  "message": "用户不存在"
}

// 系统错误：HTTP 500
{
  "code": 1,
  "message": "An internal server error occurred"
}
```

### API错误处理

```go
const (
    ErrCodeUserNotFound = 10001
    ErrCodeInvalidParam = 10002
)

func GetUser(id int) (*User, error) {
    if id <= 0 {
        return nil, errors.WithCode(ErrCodeInvalidParam, "invalid user id: %d", id)
    }
    
    user, err := userRepo.GetByID(id)
    if err == sql.ErrNoRows {
        return nil, errors.WithCode(ErrCodeUserNotFound, "user not found: %d", id)
    }
    if err != nil {
        return nil, errors.Wrap(err, "failed to get user from database")
    }
    
    return user, nil
}
```

### 批量验证

```go
func ValidateUser(user *User) error {
    var errs []error
    
    if user.Name == "" {
        errs = append(errs, errors.New("name is required"))
    }
    if user.Email == "" {
        errs = append(errs, errors.New("email is required"))
    }
    if user.Age < 0 {
        errs = append(errs, errors.New("age must be non-negative"))
    }
    
    return errors.NewAggregate(errs)
}
```

### 并发任务错误收集

```go
func ProcessFiles(files []string) error {
    funcs := make([]func() error, len(files))
    for i, file := range files {
        file := file // 避免闭包问题
        funcs[i] = func() error {
            return processFile(file)
        }
    }
    
    return errors.AggregateGoroutines(funcs...)
}
```

## 🎯 设计特点

1. **类型安全** - 基于接口设计，支持类型断言
2. **性能优化** - 高效的堆栈跟踪和错误聚合
3. **兼容性强** - 与标准库和第三方包完全兼容
4. **企业级** - 支持错误码、HTTP状态码映射
5. **可扩展** - 支持自定义错误类型和处理器

## 📊 代码统计

- **总代码量**: 1,353 行
- **核心功能**: 基础错误处理、错误码系统、聚合处理
- **依赖**: 仅依赖Go标准库
- **测试覆盖**: 包含完整的单元测试

## 🔗 相关资源

- 基于 [github.com/pkg/errors](https://github.com/pkg/errors)
- 兼容 Go 1.13+ 错误处理特性
- 适用于微服务、Web API、CLI应用等场景
