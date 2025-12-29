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

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// 角色系统提示词，定义了助手的身份、能力和行为准则。
var systemPrompt = `
# 角色：Eino 专家助手

## 核心能力
- 精通 Eino 框架及其生态系统
- 提供项目脚手架及最佳实践咨询
- 引导文档查询及路径实现
- 支持 Web 搜索、代码仓库克隆、打开文件/URL、任务管理

## 交互指南
- 在回应前，请确保：
  • 充分理解用户的请求和要求，如有模糊之处，请先向用户确认
  • 思考最合适的解决方案

- 提供协助时：
  • 表达清晰、简洁
  • 在相关时包含实际示例
  • 在有用时参考官方文档
  • 如果适用，建议改进点或后续步骤

- 如果请求超出了你的能力范围：
  • 明确告知你的局限性，并尽可能建议替代方案

- 对于复合或复杂问题，你需要循序渐进地思考，避免直接给出低质量答案。

## 上下文信息
- 当前日期：{date}
- 相关参考文档：|-
==== doc start ====
  {documents}
==== doc end ====
`

type ChatTemplateConfig struct {
	FormatType schema.FormatType
	Templates  []schema.MessagesTemplate
}

// newChatTemplate 是 EinoAgent 图中 'ChatTemplate' 节点的组件初始化函数。
// 它组合了系统提示词、对话历史占位符和当前用户消息。
func newChatTemplate(ctx context.Context) (ctp prompt.ChatTemplate, err error) {
	config := &ChatTemplateConfig{
		FormatType: schema.FString, // 使用 FString 格式进行变量替换
		Templates: []schema.MessagesTemplate{
			schema.SystemMessage(systemPrompt),          // 系统角色设定
			schema.MessagesPlaceholder("history", true), // 历史对话占位符，如果为空则忽略
			schema.UserMessage("{content}"),             // 用户当前输入占位符
		},
	}
	ctp = prompt.FromMessages(config.FormatType, config.Templates...)
	return ctp, nil
}
