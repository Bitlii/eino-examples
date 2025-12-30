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

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

// newLambda1 是 EinoAgent 图中 'ReactAgent' 节点的组件初始化函数。
// 它构建了一个 ReAct 智能体，能够根据用户意图自动查阅知识并调用工具。
func newLambda1(ctx context.Context) (lba *compose.Lambda, err error) {
	// 配置 ReAct 智能体
	config := &react.AgentConfig{
		MaxStep:            25,                    // 最大思考/执行步骤
		ToolReturnDirectly: map[string]struct{}{}, // 指定哪些工具的结果直接返回给用户
	}

	// 1. 设置聊天模型
	chatModelIns11, err := newChatModel(ctx)
	if err != nil {
		return nil, err
	}
	config.ToolCallingModel = chatModelIns11

	// 2. 注入工具集
	tools, err := GetTools(ctx)
	if err != nil {
		return nil, err
	}
	config.ToolsConfig.Tools = tools

	// 3. 创建智能体实例
	ins, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}

	// 4. 将智能体包装为 Eino Lambda 节点，支持流式和非流式输出
	lba, err = compose.AnyLambda(ins.Generate, ins.Stream, nil, nil)
	if err != nil {
		return nil, err
	}
	return lba, nil
}
