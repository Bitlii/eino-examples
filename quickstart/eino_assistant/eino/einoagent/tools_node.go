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

package einoagent

import (
	"context"

	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/tool/einotool"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/tool/gitclone"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/tool/open"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/tool/task"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/tool"
)

// GetTools 获取智能体可用的所有工具集。
func GetTools(ctx context.Context) ([]tool.BaseTool, error) {
	// 1. Eino 助手工具：提供 Eino 相关信息
	einoAssistantTool, err := NewEinoAssistantTool(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 任务管理工具：增删改查任务
	toolTask, err := NewTaskTool(ctx)
	if err != nil {
		return nil, err
	}

	// 3. 打开文件/网页工具
	toolOpen, err := NewOpenFileTool(ctx)
	if err != nil {
		return nil, err
	}

	// 4. Git 操作工具：克隆或拉取仓库
	toolGitClone, err := NewGitCloneFile(ctx)
	if err != nil {
		return nil, err
	}

	// 5. DuckDuckGo 搜索工具：联网搜索
	toolDDGSearch, err := NewDDGSearch(ctx, nil)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{
		einoAssistantTool,
		toolTask,
		toolOpen,
		toolGitClone,
		toolDDGSearch,
	}, nil
}

func defaultDDGSearchConfig(ctx context.Context) (*duckduckgo.Config, error) {
	config := &duckduckgo.Config{}
	return config, nil
}

// NewDDGSearch 创建联网搜索工具。
func NewDDGSearch(ctx context.Context, config *duckduckgo.Config) (tn tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultDDGSearchConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	tn, err = duckduckgo.NewTextSearchTool(ctx, config)
	if err != nil {
		return nil, err
	}
	return tn, nil
}

// NewOpenFileTool 创建打开文件/URL 工具。
func NewOpenFileTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return open.NewOpenFileTool(ctx, nil)
}

// NewGitCloneFile 创建 Git 克隆工具。
func NewGitCloneFile(ctx context.Context) (tn tool.BaseTool, err error) {
	return gitclone.NewGitCloneFile(ctx, nil)
}

// NewEinoAssistantTool 创建 Eino 助手工具。
func NewEinoAssistantTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return einotool.NewEinoAssistantTool(ctx, nil)
}

// NewTaskTool 创建任务管理工具。
func NewTaskTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return task.NewTaskTool(ctx, nil)
}
