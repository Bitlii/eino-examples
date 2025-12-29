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

import "github.com/cloudwego/eino/schema"

// UserMessage 定义了从客户端接收的用户消息结构。
type UserMessage struct {
	ID      string            `json:"id"`      // 消息 ID
	Query   string            `json:"query"`   // 用户查询文本
	History []*schema.Message `json:"history"` // 历史对话记录
}
