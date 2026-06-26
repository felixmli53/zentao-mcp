# 禅道 MCP 本地修改记录

> 本文档记录了相对于官方仓库 `zentao-mcp`（commit `d5ea85f`）的本地未提交修改。
> 当官方发布新版本需要合并时，以本文档为参照处理冲突。

---

## 问题背景

禅道 OpenAPI spec 大量详情端点（如 `/tasks/:taskID`、`/executions/:executionID/tasks`）只在 URL 路径模板中写了参数占位符，却没有在 `operation.parameters` 中声明对应的 path parameter。

官方代码在构建 MCP 工具时只遍历 `parameters` 列表，导致：
1. **输入 schema 缺失参数** — AI 不知道需要传 `taskID`，无法正确调用
2. **请求 URL 占位符未替换** — 执行时 `:taskID` 原样保留在 URL 中，禅道返回 404 或空数据

这是一个 **"200 但拿不到数据"** 的典型原因：请求打到了错误的路径上。

---

## 本地改动详情

### 1. 路径参数自动推断 — `extractPathParamNames`

**文件**：`internal/service/schema/build_input.go`

**新增内容**：
- 全局正则 `rePathParam`：同时匹配 `:name` 和 `{name}` 两种路径参数格式
- `buildToolInputSchema` 签名新增 `pathTmpl string` 参数
- 在遍历完 spec 定义的参数后，额外扫描 URL 模板：
  1. 收集 spec 中已声明的 path 参数 → `definedPathParams`
  2. 用 `extractPathParamNames()` 提取模板中的参数名
  3. 对未在 spec 中声明的参数，自动补为 `type: string` + `required`

**为什么推断为 `string` 而非 `integer`**：
禅道的路径参数既有纯数字 ID（如 `taskID=123`），也有字符串代号（如 `projectCode=my-project`）。如果硬编码为 `integer`，字符串参数会导致 404。`string` 更安全，因为数字值传给 string 参数不会有问题，反过来则不行。

```go
// 自动推断的参数定义
prop := &openapi3.Schema{
    Type:        &openapi3.Types{"string"},
    Description: fmt.Sprintf("Path parameter: %s", name),
}
```

**合并注意**：
- 函数签名变了（新增第 5 个参数 `pathTmpl`），调用方必须同步修改
- `extractPathParamNames` 和 `rePathParam` 是新增的，合并时直接保留即可
- 如果上游也做了类似的参数推断，注意去重逻辑不要重复补参数

---

### 2. `collectParams` 同步补全路径参数

**文件**：`internal/service/schema/tools.go`

**新增内容**：
- `collectParams` 签名新增 `pathTmpl string`
- 逻辑与 `buildToolInputSchema` 中的推断完全对称：
  1. 收集已声明的 path 参数 → `definedPathParams`
  2. 用 `extractPathParamNames()` 扫描模板
  3. 补全缺失的 path 参数到 `ToolParam` 列表

**目的**：`collectParams` 的结果用于 `Execute()` 中构建请求 URL 和查询参数。如果这里不加推断，即使输入 schema 补了参数，执行时也不会把参数值填到 URL 里。

**合并注意**：两处推断逻辑共用 `extractPathParamNames` 和 `rePathParam`，必须保持一致。

---

### 3. `sanitizePathName` 增强

**文件**：`internal/service/schema/tools.go`

**改动**：
- 新增 `strings.ReplaceAll(s, ":", "")` — 去掉冒号
- 将原来的 `strings.ReplaceAll(s, "-", "_")` 替换为 `invalidToolNameChars.ReplaceAllString(s, "_")` — 所有非字母数字下划线字符都替换为下划线
- 新增 `strings.Trim(s, "_")` — 去掉首尾下划线

**目的**：禅道路径含冒号（如 `/tasks/:id`），旧代码生成的工具名会包含非法字符 `:`。

**合并注意**：这是一个 **破坏性变更** — 旧版本生成的工具名（如 `get_tasks_:id`）在新版本变为 `get_tasks_id`。升级后客户端缓存的旧工具名会失效，需要重新发现工具。

---

### 4. `.gitignore` 增加 `zentao-mcp`

**文件**：`.gitignore`

忽略本地编译产物。

---

## 改动文件清单

| 文件 | 改动要点 |
|------|---------|
| `internal/service/schema/build_input.go` | 新增 `rePathParam`、`extractPathParamNames`；`buildToolInputSchema` 签名新增 `pathTmpl` 并自动推断缺失路径参数 |
| `internal/service/schema/tools.go` | `buildToolInputSchema` 调用传入 `p`；`collectParams` 签名新增 `pathTmpl` 并推断缺失路径参数；`sanitizePathName` 增强处理冒号和首尾下划线 |
| `internal/service/schema/build_input_test.go` | 测试调用适配新签名（传入 `""`） |
| `.gitignore` | 忽略 `zentao-mcp` 编译产物 |
