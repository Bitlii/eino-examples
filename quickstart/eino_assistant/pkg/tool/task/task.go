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

// Package task 提供了任务管理工具的 Eino 集成。
package task

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/google/uuid"
)

// Action 定义了任务工具支持的操作类型。
type Action string

const (
	ActionAdd    Action = "add"    // 添加任务
	ActionGet    Action = "get"    // 获取任务（注：当前实现在 Invoke 中未特别处理，通常归类为查询或通过 ID 获取）
	ActionUpdate Action = "update" // 更新任务
	ActionDelete Action = "delete" // 删除任务
	ActionList   Action = "list"   // 列出任务
)

// Task 定义了任务的数据结构，支持 JSON 序列化和 JSON Schema 描述。
type Task struct {
	ID        string `json:"id" jsonschema_description:"任务的唯一标识符"`
	Title     string `json:"title" jsonschema_description:"任务标题"`
	Content   string `json:"content" jsonschema_description:"任务详细内容"`
	Completed bool   `json:"completed" jsonschema_description:"任务是否已完成"`
	Deadline  string `json:"deadline" jsonschema_description:"任务截止日期"`
	IsDeleted bool   `json:"is_deleted" jsonschema:"-"` // 软删除标记，不公开给 LLM

	CreatedAt string `json:"created_at" jsonschema_description:"任务创建时间"`
}

// TaskRequest 定义了调用任务管理工具的请求体。
type TaskRequest struct {
	Action Action      `json:"action" jsonschema_description:"要执行的操作，可选值：add, update, delete, list"`
	Task   *Task       `json:"task" jsonschema_description:"要添加、更新或删除的任务对象"`
	List   *ListParams `json:"list" jsonschema_description:"列出任务时的过滤和分页参数"`
}

// ListParams 定义了列出任务时的过滤条件。
type ListParams struct {
	Query  string `json:"query" jsonschema_description:"用于搜索标题或内容的关键字"`
	IsDone *bool  `json:"is_done" jsonschema_description:"按完成状态过滤"`
	Limit  *int   `json:"limit" jsonschema_description:"限制返回的结果数量"`
}

// TaskResponse 定义了任务管理工具的响应体。
type TaskResponse struct {
	Status string `json:"status" jsonschema_description:"操作结果状态，如 'success' 或 'error'"`

	TaskList []*Task `json:"task_list" jsonschema_description:"返回的任务列表（用于 list 或 add/update 操作）"`

	Error string `json:"error" jsonschema_description:"错误信息，如果操作成功则为空"`
}

// TaskToolImpl 是任务管理工具的实现。
type TaskToolImpl struct {
	config *TaskToolConfig
}

// TaskToolConfig 包含了任务工具的配置，主要是持久化存储引擎。
type TaskToolConfig struct {
	Storage *Storage
}

func defaultTaskToolConfig(ctx context.Context) (*TaskToolConfig, error) {
	config := &TaskToolConfig{
		Storage: GetDefaultStorage(),
	}
	return config, nil
}

// NewTaskToolImpl 创建一个新的 TaskToolImpl 实例。
func NewTaskToolImpl(ctx context.Context, config *TaskToolConfig) (*TaskToolImpl, error) {
	var err error
	if config == nil {
		config, err = defaultTaskToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	if config.Storage == nil {
		return nil, fmt.Errorf("存储引擎不能为空")
	}

	t := &TaskToolImpl{config: config}

	return t, nil
}

// NewTaskTool 创建一个满足 Eino tool.BaseTool 接口的任务管理工具。
func NewTaskTool(ctx context.Context, config *TaskToolConfig) (tn tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultTaskToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	if config.Storage == nil {
		return nil, fmt.Errorf("存储引擎不能为空")
	}

	t := &TaskToolImpl{config: config}
	tn, err = t.ToEinoTool()
	if err != nil {
		return nil, err
	}
	return tn, nil
}

// ToEinoTool 将实现转换为 Eino 框架可识别的工具接口。
func (t *TaskToolImpl) ToEinoTool() (tool.BaseTool, error) {
	return utils.InferTool("task_manager", "任务管理工具，用于添加、更新、删除和列出个人任务", t.Invoke)
}

// Invoke 是工具执行的入口函数，根据 Action 执行不同逻辑。
func (t *TaskToolImpl) Invoke(ctx context.Context, req *TaskRequest) (res *TaskResponse, err error) {
	res = &TaskResponse{}

	switch req.Action {
	case ActionAdd:
		if req.Task == nil {
			res.Status = "error"
			res.Error = "添加操作需要提供任务详情"
			return res, nil
		}
		if req.Task.Title == "" {
			res.Status = "error"
			res.Error = "任务标题是必需的"
			return res, nil
		}
		// 生成唯一 ID 并存储
		req.Task.ID = uuid.New().String()
		if err := t.config.Storage.Add(req.Task); err != nil {
			res.Status = "error"
			res.Error = fmt.Sprintf("添加任务失败: %v", err)
			return res, nil
		}
		res.TaskList = []*Task{req.Task}

	case ActionUpdate:
		if req.Task == nil {
			res.Status = "error"
			res.Error = "更新操作需要提供任务详情"
			return res, nil
		}
		if req.Task.ID == "" {
			res.Status = "error"
			res.Error = "任务 ID 是必需的"
			return res, nil
		}
		if err := t.config.Storage.Update(req.Task); err != nil {
			res.Status = "error"
			res.Error = fmt.Sprintf("更新任务失败: %v", err)
			return res, nil
		}
		res.TaskList = []*Task{req.Task}

	case ActionDelete:
		if req.Task == nil || req.Task.ID == "" {
			res.Status = "error"
			res.Error = "删除操作需要提供任务 ID"
			return res, nil
		}
		if err := t.config.Storage.Delete(req.Task.ID); err != nil {
			res.Status = "error"
			res.Error = fmt.Sprintf("删除任务失败: %v", err)
			return res, nil
		}

	case ActionList:
		if req.List == nil {
			req.List = &ListParams{}
		}
		tasks, err := t.config.Storage.List(req.List)
		if err != nil {
			res.Status = "error"
			res.Error = fmt.Sprintf("获取任务列表失败: %v", err)
			return res, nil
		}
		res.TaskList = tasks

	default:
		res.Status = "error"
		res.Error = fmt.Sprintf("未知的操作类型: %s", req.Action)
	}

	res.Status = "success"
	return res, nil
}
