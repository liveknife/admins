// Package docs 生成并暴露 OpenAPI 3.0 文档 + Swagger UI。
// 设计目标：不依赖 swag CLI、不写生成物；路由 → 文档描述集中在 routes.go
// 数据结构 → schema 使用反射自动生成，避免手写 JSON schema 的重复。
package docs

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ──────────────────────────────────────────────
// DSL：描述一条路由的文档
// ──────────────────────────────────────────────

type Op struct {
	Summary     string   // 简介，Swagger UI 每行显示
	Description string   // 详细说明，支持 Markdown
	Tags        []string // 分组标签
	Security    bool     // 是否需要 Bearer JWT
	Permission  string   // 需要的权限码；会拼进描述
	Params      []Param  // path/query/header 参数
	Body        Body     // 请求体（json）
	Responses   []Resp   // 响应；不写则默认 200 空对象
	Deprecated  bool
}

type Param struct {
	Name        string
	In          string // path / query / header
	Description string
	Required    bool
	Type        string // string / integer / boolean / number；留空默认 string
	Example     any
}

type Body struct {
	Description string
	Required    bool
	Schema      any // Go 类型或 SchemaRef("Name")
}

type Resp struct {
	Status      string // "200"、"201"、"default" 等；默认 "200"
	Description string
	Schema      any // Go 类型 / SchemaRef / map / 内联 map[string]any
}

// SchemaRef 用于引用已注册的 components/schemas
type SchemaRef string

// Route 是 routes.go 里的每条路由项
type Route struct {
	Method string
	Path   string // Gin 风格：/api/v1/users/:id
	Op     Op
}

// ──────────────────────────────────────────────
// Schema 注册表：反射 Go struct → JSON Schema
// ──────────────────────────────────────────────

type schemaRegistry struct {
	mu      sync.Mutex
	byName  map[string]map[string]any
	byType  map[reflect.Type]string
	visitAt map[reflect.Type]bool // 防循环
}

func newSchemaRegistry() *schemaRegistry {
	return &schemaRegistry{
		byName:  map[string]map[string]any{},
		byType:  map[reflect.Type]string{},
		visitAt: map[reflect.Type]bool{},
	}
}

var timeType = reflect.TypeOf(time.Time{})

// register 递归注册 t 及其所有嵌套 struct，返回顶层 schema 名称
func (r *schemaRegistry) register(t reflect.Type) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if name, ok := r.byType[t]; ok {
		return name
	}
	if t.Kind() != reflect.Struct || t == timeType {
		return ""
	}
	name := schemaName(t)
	r.byType[t] = name
	if r.visitAt[t] {
		r.byName[name] = map[string]any{"type": "object"}
		return name
	}
	r.visitAt[t] = true
	r.byName[name] = r.buildObjectSchema(t)
	return name
}

func (r *schemaRegistry) buildObjectSchema(t reflect.Type) map[string]any {
	props := map[string]any{}
	required := []string{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		jsonTag := f.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		fieldName, opts := parseJSONTag(jsonTag, f.Name)
		schema := r.typeSchema(f.Type)
		if desc := f.Tag.Get("desc"); desc != "" {
			schema["description"] = desc
		}
		if example := f.Tag.Get("example"); example != "" {
			schema["example"] = example
		}
		props[fieldName] = schema
		if bindingHasRequired(f.Tag.Get("binding")) && !strings.Contains(opts, "omitempty") {
			required = append(required, fieldName)
		}
	}
	out := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func (r *schemaRegistry) typeSchema(t reflect.Type) map[string]any {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == timeType {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return map[string]any{"type": "integer", "format": "int32"}
	case reflect.Int64, reflect.Uint64:
		return map[string]any{"type": "integer", "format": "int64"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number", "format": "double"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": r.typeSchema(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": r.typeSchema(t.Elem())}
	case reflect.Struct:
		name := r.register(t)
		if name == "" {
			return map[string]any{"type": "object"}
		}
		return map[string]any{"$ref": "#/components/schemas/" + name}
	case reflect.Interface:
		return map[string]any{}
	}
	return map[string]any{"type": "string"}
}

func schemaName(t reflect.Type) string {
	// 取 pkg 最后一段 + 类型名，避免不同包同名冲突
	name := t.Name()
	if name == "" {
		return "Object"
	}
	pkg := t.PkgPath()
	if idx := strings.LastIndex(pkg, "/"); idx >= 0 {
		pkg = pkg[idx+1:]
	}
	if pkg == "" || pkg == "models" {
		return name
	}
	return pkg + "_" + name
}

func parseJSONTag(tag, fallback string) (name, opts string) {
	if tag == "" {
		return fallback, ""
	}
	parts := strings.SplitN(tag, ",", 2)
	name = parts[0]
	if name == "" {
		name = fallback
	}
	if len(parts) == 2 {
		opts = parts[1]
	}
	return
}

func bindingHasRequired(binding string) bool {
	for _, part := range strings.Split(binding, ",") {
		if strings.TrimSpace(part) == "required" {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────
// 生成 OpenAPI 3.0 文档
// ──────────────────────────────────────────────

type Info struct {
	Title       string
	Version     string
	Description string
	Schemas     []any // 预先注册的 Go 类型；确保 SchemaRef("Name") 能找到对应 schema
	Lang        string // 空 或 "zh" 保持原文；"en" 使用内置字典翻译
}

// translator 返回文本的目标语言版本。找不到翻译时返回原文。
type translator func(string) string

func identity(s string) string { return s }

func translatorFor(lang string) translator {
	if strings.EqualFold(lang, "en") {
		return translate
	}
	return identity
}

func Build(info Info, routes []Route) map[string]any {
	tr := translatorFor(info.Lang)
	reg := newSchemaRegistry()
	for _, seed := range info.Schemas {
		reg.typeSchema(reflect.TypeOf(seed))
	}
	paths := map[string]map[string]any{}
	tagSet := map[string]bool{}

	for _, r := range routes {
		path := ginPathToOpenAPI(r.Path)
		if _, ok := paths[path]; !ok {
			paths[path] = map[string]any{}
		}
		paths[path][strings.ToLower(r.Method)] = buildOperation(reg, r, tr)
		for _, t := range r.Op.Tags {
			tagSet[t] = true
		}
	}

	tags := make([]map[string]any, 0, len(tagSet))
	for t := range tagSet {
		tags = append(tags, map[string]any{"name": tr(t)})
	}
	// 保持稳定顺序
	sortStableStringSlice(tags, func(m map[string]any) string { return m["name"].(string) })

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       info.Title,
			"version":     info.Version,
			"description": tr(info.Description),
		},
		"tags":  tags,
		"paths": paths,
		"components": map[string]any{
			"schemas": reg.byName,
			"securitySchemes": map[string]any{
				"BearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
	}
}

func buildOperation(reg *schemaRegistry, r Route, tr translator) map[string]any {
	translatedTags := make([]string, 0, len(r.Op.Tags))
	for _, t := range r.Op.Tags {
		translatedTags = append(translatedTags, tr(t))
	}
	op := map[string]any{"summary": tr(r.Op.Summary), "tags": translatedTags}
	desc := strings.TrimSpace(tr(r.Op.Description))
	if r.Op.Permission != "" {
		if desc != "" {
			desc += "\n\n"
		}
		// "需要权限：" 也走字典
		desc += tr("需要权限：") + "`" + r.Op.Permission + "`"
	}
	if desc != "" {
		op["description"] = desc
	}
	if r.Op.Deprecated {
		op["deprecated"] = true
	}
	if r.Op.Security {
		op["security"] = []map[string]any{{"BearerAuth": []string{}}}
	}
	if params := buildParams(r, tr); len(params) > 0 {
		op["parameters"] = params
	}
	if body := buildRequestBody(reg, r.Op.Body, tr); body != nil {
		op["requestBody"] = body
	}
	op["responses"] = buildResponses(reg, r.Op.Responses, tr)
	return op
}

func buildParams(r Route, tr translator) []map[string]any {
	out := []map[string]any{}
	seen := map[string]bool{}
	// path 参数根据 :name 自动补齐（Op.Params 里描述则合并）
	for _, seg := range strings.Split(r.Path, "/") {
		if !strings.HasPrefix(seg, ":") {
			continue
		}
		name := strings.TrimPrefix(seg, ":")
		seen[name] = true
		desc, typ := "", "string"
		for _, p := range r.Op.Params {
			if p.In == "path" && p.Name == name {
				desc = p.Description
				if p.Type != "" {
					typ = p.Type
				}
			}
		}
		out = append(out, map[string]any{
			"name": name, "in": "path", "required": true,
			"description": tr(desc),
			"schema":      map[string]any{"type": typ},
		})
	}
	for _, p := range r.Op.Params {
		if p.In == "path" && seen[p.Name] {
			continue
		}
		typ := p.Type
		if typ == "" {
			typ = "string"
		}
		item := map[string]any{
			"name": p.Name, "in": p.In, "required": p.Required,
			"description": tr(p.Description),
			"schema":      map[string]any{"type": typ},
		}
		if p.Example != nil {
			item["example"] = p.Example
		}
		out = append(out, item)
	}
	return out
}

func buildRequestBody(reg *schemaRegistry, b Body, tr translator) map[string]any {
	if b.Schema == nil {
		return nil
	}
	return map[string]any{
		"description": tr(b.Description),
		"required":    b.Required,
		"content": map[string]any{
			"application/json": map[string]any{"schema": schemaFor(reg, b.Schema)},
		},
	}
}

func buildResponses(reg *schemaRegistry, resps []Resp, tr translator) map[string]any {
	if len(resps) == 0 {
		return map[string]any{"200": map[string]any{"description": "OK"}}
	}
	out := map[string]any{}
	for _, r := range resps {
		status := r.Status
		if status == "" {
			status = "200"
		}
		item := map[string]any{"description": tr(defaultDescription(status, r.Description))}
		if r.Schema != nil {
			item["content"] = map[string]any{
				"application/json": map[string]any{"schema": schemaFor(reg, r.Schema)},
			}
		}
		out[status] = item
	}
	return out
}

func defaultDescription(status, desc string) string {
	if desc != "" {
		return desc
	}
	switch status {
	case "200":
		return "OK"
	case "201":
		return "Created"
	case "204":
		return "No Content"
	case "400":
		return "Bad Request"
	case "401":
		return "Unauthorized"
	case "403":
		return "Forbidden"
	case "404":
		return "Not Found"
	case "409":
		return "Conflict"
	case "429":
		return "Too Many Requests"
	case "500":
		return "Internal Server Error"
	}
	return "Response"
}

// schemaFor 支持三种形式：Go 类型 (reflect.Type/reflect.Value)、SchemaRef、任意 map[string]any
func schemaFor(reg *schemaRegistry, spec any) map[string]any {
	switch v := spec.(type) {
	case nil:
		return map[string]any{}
	case SchemaRef:
		return map[string]any{"$ref": "#/components/schemas/" + string(v)}
	case map[string]any:
		return v
	case reflect.Type:
		return reg.typeSchema(v)
	}
	// 传入的是一个 Go 值实例
	return reg.typeSchema(reflect.TypeOf(spec))
}

func ginPathToOpenAPI(path string) string {
	if !strings.Contains(path, ":") {
		return path
	}
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") {
			parts[i] = "{" + strings.TrimPrefix(p, ":") + "}"
		}
	}
	return strings.Join(parts, "/")
}

func sortStableStringSlice(items []map[string]any, key func(map[string]any) string) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && key(items[j]) < key(items[j-1]); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

// ──────────────────────────────────────────────
// 挂载：/swagger/doc.json + /swagger/index.html
// ──────────────────────────────────────────────

// Register 挂载 Swagger UI 与 OpenAPI JSON 到 Gin 引擎。
// mountPath 默认 "/swagger"；info 里 Title/Version 必填。
// 支持中英切换：
//   - /swagger/doc.json           → 默认（中文）
//   - /swagger/doc.json?lang=en   → 英文（内置字典翻译，命中率不足时保底原文）
//   - /swagger/doc.json?lang=zh   → 中文
// UI 顶部有 "中文 / EN" 切换按钮。
func Register(r *gin.Engine, mountPath string, info Info, routes []Route) {
	if mountPath == "" {
		mountPath = "/swagger"
	}
	// 提前构建两种语言的 spec —— 生成一次，之后 O(1) 响应
	if info.Lang == "" {
		info.Lang = "zh"
	}
	zhInfo := info
	zhInfo.Lang = "zh"
	enInfo := info
	enInfo.Lang = "en"
	zhJSON, _ := json.Marshal(Build(zhInfo, routes))
	enJSON, _ := json.Marshal(Build(enInfo, routes))

	r.GET(mountPath+"/doc.json", func(c *gin.Context) {
		if strings.EqualFold(c.Query("lang"), "en") {
			c.Data(http.StatusOK, "application/json; charset=utf-8", enJSON)
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", zhJSON)
	})
	r.GET(mountPath, func(c *gin.Context) {
		c.Redirect(http.StatusFound, mountPath+"/index.html")
	})
	r.GET(mountPath+"/index.html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(renderUI(mountPath, info.Title)))
	})
}
