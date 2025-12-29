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

// Package einoagent 实现了 Eino 助手的核心智能逻辑。
package einoagent

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// BuildEinoAgent 构建并编译 Eino 助手的执行图。
// 流程：开始 -> (InputToQuery, InputToHistory) -> RedisRetriever -> ChatTemplate -> ReactAgent -> 结束。
func BuildEinoAgent(ctx context.Context) (r compose.Runnable[*UserMessage, *schema.Message], err error) {
	// 定义图中的节点 Key
	const (
		InputToQuery   = "InputToQuery"   // 解析输入为查询语句
		ChatTemplate   = "ChatTemplate"   // 组装聊天模板
		ReactAgent     = "ReactAgent"     // ReAct 智能体节点
		RedisRetriever = "RedisRetriever" // 知识库检索节点
		InputToHistory = "InputToHistory" // 解析输入为历史变量
	)

	// 创建图：输入为 UserMessage，输出为回复消息
	g := compose.NewGraph[*UserMessage, *schema.Message]()

	// 1. 添加各个处理节点
	_ = g.AddLambdaNode(InputToQuery, compose.InvokableLambdaWithOption(newLambda), compose.WithNodeName("用户输入转查询"))

	chatTemplateKeyOfChatTemplate, err := newChatTemplate(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddChatTemplateNode(ChatTemplate, chatTemplateKeyOfChatTemplate)

	reactAgentKeyOfLambda, err := newLambda1(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddLambdaNode(ReactAgent, reactAgentKeyOfLambda, compose.WithNodeName("ReAct 智能体"))

	redisRetrieverKeyOfRetriever, err := newRetriever(ctx)
	if err != nil {
		return nil, err
	}
	// 检索输出会放入 "documents" 键中供后继节点使用
	_ = g.AddRetrieverNode(RedisRetriever, redisRetrieverKeyOfRetriever, compose.WithOutputKey("documents"))

	_ = g.AddLambdaNode(InputToHistory, compose.InvokableLambdaWithOption(newLambda2), compose.WithNodeName("用户输入转变量"))

	// 2. 连接节点编排流程
	_ = g.AddEdge(compose.START, InputToQuery)
	_ = g.AddEdge(compose.START, InputToHistory)
	_ = g.AddEdge(InputToQuery, RedisRetriever)
	_ = g.AddEdge(RedisRetriever, ChatTemplate)
	_ = g.AddEdge(InputToHistory, ChatTemplate)
	_ = g.AddEdge(ChatTemplate, ReactAgent)
	_ = g.AddEdge(ReactAgent, compose.END)

	// 3. 编译图，所有前置节点完成后触发后续节点 (AllPredecessor)
	r, err = g.Compile(ctx, compose.WithGraphName("EinoAgent"), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		return nil, err
	}
	return r, err
}
