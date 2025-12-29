/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package einotool 提供了用于获取 Eino 相关信息和初始化项目模板的工具。
package einotool

import (
	"context"
	"embed"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

//go:embed templates/*
var templateFS embed.FS

const desc = `eino 工具可以获取 eino 项目信息。
支持的操作：
- get_example_project: 获取示例项目的 URL，来自 eino-examples
- get_github_repo: 获取 GitHub 仓库地址，例如 eino, eino-ext, eino-examples
- get_doc_url: 获取 Eino 官网的文档地址
- init_template: 初始化 Eino 项目模板，从模板创建文件
`

// EinoAssistantToolImpl 是 Eino 助手工具的实现。
type EinoAssistantToolImpl struct {
	config *EinoAssistantToolConfig
}

// EinoAssistantToolConfig 包含了 Eino 助手工具的配置。
type EinoAssistantToolConfig struct {
	BaseDir string // 模板初始化的基础目录
}

func defaultEinoAssistantToolConfig(ctx context.Context) (*EinoAssistantToolConfig, error) {
	config := &EinoAssistantToolConfig{
		BaseDir: "./data/eino",
	}
	return config, nil
}

// NewEinoAssistantTool 创建一个新的 Eino 助手工具实例。
func NewEinoAssistantTool(ctx context.Context, config *EinoAssistantToolConfig) (tn tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultEinoAssistantToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	t := &EinoAssistantToolImpl{config: config}
	tn, err = t.ToEinoTool()
	if err != nil {
		return nil, err
	}
	return tn, nil
}

var (
	// EinoRepo 存储了 Eino 相关的 GitHub 仓库地址
	EinoRepo = map[string]string{
		"eino":          "https://github.com/cloudwego/eino",
		"eino-ext":      "https://github.com/cloudwego/eino-ext",
		"eino-examples": "https://github.com/cloudwego/eino-examples",
	}

	// EinoDoc 存储了 Eino 相关的文档地址
	EinoDoc = map[string]string{
		"eino_index": "https://www.cloudwego.io/zh/docs/eino/",
		"quickstart": "https://www.cloudwego.io/zh/docs/eino/quick_start/",
		"graph":      "https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/",
		"agent":      "https://www.cloudwego.io/zh/docs/eino/core_modules/flow_integration_components/",
		"components": "https://www.cloudwego.io/zh/docs/eino/core_modules/components/",
		"integrate":  "https://www.cloudwego.io/zh/docs/eino/ecosystem_integration/",
	}

	// EinoExample 存储了 Eino 示例项目的地址
	EinoExample = map[string][]string{
		"agent":      {"https://github.com/cloudwego/eino-examples/tree/main/flow/agent/react"},
		"components": {"https://github.com/cloudwego/eino-examples/tree/main/components"},
		"graph":      {"https://github.com/cloudwego/eino-examples/tree/main/compose/graph/tool_call_agent.go"},
		"quickstart": {"https://github.com/cloudwego/eino-examples/tree/main/quickstart"},
	}

	// Template 定义了初始化模板时包含的文件列表
	Template = map[string][]string{
		"react_agent": {"react_agent/main.go"},
		"simple_llm":  {"simple_llm/main.go"},
		"http_agent":  {"http_agent/main.go", "http_agent/README.md", "http_agent/client/main.go"},
	}
)

// ToEinoTool 将实现转换为 Eino 框架可识别的工具接口。
func (e *EinoAssistantToolImpl) ToEinoTool() (tool.BaseTool, error) {
	return utils.InferTool("eino_tool", desc, e.Invoke)
}

// Invoke 是工具执行的入口函数。
func (e *EinoAssistantToolImpl) Invoke(ctx context.Context, req *EinoToolRequest) (res *EinoToolResponse, err error) {
	res = &EinoToolResponse{}

	switch req.Action {
	case EinoToolActionGetExampleProject:
		exampleURL := EinoExample[req.ExampleType]
		if len(exampleURL) == 0 {
			res.Error = "无效的示例类型，可选值：agent, components, graph, quickstart。示例仓库地址为 " + EinoRepo["eino-examples"]
			return
		}
		res.Message = exampleURL[0]
	case EinoToolActionGetGithubRepo:
		repoURL := EinoRepo[req.RepoType]
		if repoURL == "" {
			res.Error = "无效的仓库类型，可选值：eino, eino-ext, eino-examples。Eino 仓库地址为 " + EinoRepo["eino"]
			return
		}
		res.Message = repoURL
	case EinoToolActionGetDocURL:
		docURL := EinoDoc[req.DocType]
		if docURL == "" {
			res.Error = "无效的文档类型，可选值：eino_index, quickstart, graph, agent, components, integrate。Eino 主页文档地址为 " + EinoDoc["eino_index"]
			return
		}
		res.Message = docURL
	case EinoToolActionInitTemplate:
		templateURL := Template[req.TemplateType]
		if len(templateURL) == 0 {
			res.Error = "无效的模板类型，可选值：react_agent, simple_llm, http_agent"
			return res, nil
		}

		baseDir := e.config.BaseDir
		for _, file := range templateURL {
			// 读取模板文件
			content, err := templateFS.ReadFile(filepath.Join("templates", file))
			if err != nil {
				res.Error = "读取模板文件失败: " + err.Error()
				return res, nil
			}

			// 创建目标目录
			targetPath := filepath.Join(baseDir, file)
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				res.Error = "创建目录失败: " + err.Error()
				return res, nil
			}

			// 写入文件
			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				res.Error = "写入文件失败: " + err.Error()
				return res, nil
			}
		}
		absPath, err := filepath.Abs(filepath.Join(baseDir, req.TemplateType))
		if err != nil {
			absPath = filepath.Join(baseDir, req.TemplateType)
		}
		res.Message = "成功初始化模板，路径为: " + absPath
		return res, nil
	default:
		res.Error = "无效的操作，可选值：get_example_project, get_github_repo, get_doc_url, init_template"
	}

	return res, nil
}

// EinoToolAction 定义了工具支持的操作类型。
type EinoToolAction string

const (
	EinoToolActionGetExampleProject EinoToolAction = "get_example_project" // 获取示例项目
	EinoToolActionGetGithubRepo     EinoToolAction = "get_github_repo"     // 获取 GitHub 仓库
	EinoToolActionGetDocURL         EinoToolAction = "get_doc_url"         // 获取文档地址
	EinoToolActionInitTemplate      EinoToolAction = "init_template"       // 初始化项目模板
)

// EinoToolRequest 定义了工具的请求参数。
type EinoToolRequest struct {
	Action       EinoToolAction `json:"action" jsonschema_description:"请求的操作类型，枚举值：get_example_project, get_github_repo, get_doc_url, init_template"`
	ExampleType  string         `json:"example_type,omitempty" jsonschema_description:"示例项目的类型，仅在 action 为 get_example_project 时有效。枚举值：agent, components, graph, quickstart"`
	RepoType     string         `json:"repo_type,omitempty" jsonschema_description:"仓库类型，仅在 action 为 get_github_repo 时有效。枚举值：eino, eino-ext, eino-examples"`
	DocType      string         `json:"doc_type,omitempty" jsonschema_description:"文档类型，仅在 action 为 get_doc_url 时有效。枚举值：eino_index, quickstart, graph, agent, components, integrate"`
	TemplateType string         `json:"template_type,omitempty" jsonschema_description:"项目模板类型，仅在 action 为 init_template 时有效。枚举值：react_agent, simple_llm, http_agent"`
}

// EinoToolResponse 定义了工具的响应结果。
type EinoToolResponse struct {
	Message string `json:"message" jsonschema_description:"响应的消息内容"`
	Error   string `json:"error" jsonschema_description:"错误信息，如果执行成功则为空"`
}
