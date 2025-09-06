# ForgeC Codex 工作清单

下面是给 **Codex** 的一份完整工作清单，逐步票据化，确保它能够实现出最小可用版的 **ForgeC**（Go→C 导出生成器）。

---

## 总目标
实现一个名为 **ForgeC** 的 Go 代码生成器（CLI 工具），能：
1. 扫描 `./internal` 包中带有 `capi:export` 注释的**顶层函数**；
2. 验证函数签名符合原型规则：`func(...int32) (int32, error)`；
3. 生成 `exports.go`（含 `//export` 符号、参数封送、errno 返回、Sentry panic 捕获、`capi_free` 与 `capi_last_error_json`）；
4. 生成配套头文件 `forgec.h`；
5. 提供示例项目 `examples/myapi`，可 `go generate` + `go build -buildmode=c-shared` 产出 DLL/SO/Dylib 并通过一个小 C 程序调用 `PM_Add` 成功。

---

## 票据清单

### 票据 #0: 目录搭建
- 初始化 mono-repo 与两个 module：`forgec` 生成器 + `examples/myapi` 示例。
- 验收：`go mod tidy` 无报错。

### 票据 #1: CLI 生成器骨架
- 实现 `cmd/forgec` 的 CLI 参数与主流程。
- 验收：`go run ./forgec/cmd/forgec -h` 正常。

### 票据 #2: 签名校验与参数收集
- 校验签名：必须是 `func(...int32) (int32, error)`。
- 收集参数名与类型。

### 票据 #3: 代码生成 exports.go
- 生成 `exports.go`，包含：
  - Sentry 捕获、errno 风格、last error JSON。
  - 每个函数包装 `//export`。

### 票据 #4: 代码生成 forgec.h
- 生成 `forgec.h`，包含：
  - 每个函数的 C 声明。
  - 工具函数声明：`capi_free` 和 `capi_last_error_json`。

### 票据 #5: 示例项目
- `examples/myapi/internal/calc.go`：实现 `Add` 并标注 `capi:export`。
- `examples/myapi/sentrywrap/sentrywrap.go`：提供 `RecoverAndReport`。

### 票据 #6: go generate 集成与构建
- 在 `examples/myapi/generate.go` 添加 `//go:generate`。
- 运行 `go generate ./...` 后，`go build -buildmode=c-shared` 可成功。

### 票据 #7: C 侧冒烟测试
- 编写 `examples/myapi/c_smoke.c` 调用 `PM_Add`。
- 验收：运行输出 `PM_Add(3,4)=7`。

### 票据 #8: 代码质量与幂等性
- 使用 `go/format` 格式化生成文件。
- 幂等写入。

### 票据 #9: 开发脚本与 README
- 提供仓库级 `README.md`，包含快速开始与构建说明。

---

## 一次性 Codex 提示词

> 你是一个资深 Go 构建与代码生成工程师。  
> 按以下票据逐步实现一个名为 ForgeC 的 Go→C 导出生成器。  
> 对每个票据：创建/修改文件，给出完整代码，确保可编译，并附带运行命令与期望输出。  
> 要求：
> - 生成器模块：`forgec`（`example.com/forgec`），CLI 位于 `forgec/cmd/forgec`  
> - 示例项目：`examples/myapi`（`example.com/myapi`）  
> - 解析带有 `capi:export` 注释的顶层函数，签名必须 `func(...int32) (int32, error)`  
> - 生成 `exports.go` 与 `forgec.h`（errno 风格、`capi_free`、`capi_last_error_json`、`sentrywrap.RecoverAndReport`）  
> - `go generate` 集成：`//go:generate go run example.com/forgec/cmd/forgec -pkg ./internal -o ./exports.go -hout ./forgec.h -mod example.com/myapi`  
> - 能 `go build -buildmode=c-shared` 构建共享库，并通过 C 程序成功调用 `PM_Add`  
> - 保证可在 Windows/Linux/macOS 上编译（默认 x64），使用定宽整型  
> 
> 以下为票据：
> - 票据 #0 到 #9 （见上文）
