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

// Package mem 提供简单的内存管理功能，用于存储和检索对话上下文。
package mem

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cloudwego/eino/schema"
)

// GetDefaultMemory 返回一个默认配置的 SimpleMemory 实例。
// 默认存储路径为 "data/memory"，最大窗口大小为 6。
func GetDefaultMemory() *SimpleMemory {
	return NewSimpleMemory(SimpleMemoryConfig{
		Dir:           "data/memory",
		MaxWindowSize: 6,
	})
}

// SimpleMemoryConfig 定义了 SimpleMemory 的配置参数。
type SimpleMemoryConfig struct {
	Dir           string // 存储对话记录的目录
	MaxWindowSize int    // 获取消息时的最大窗口大小（最近的 N 条消息）
}

// NewSimpleMemory 根据配置创建一个新的 SimpleMemory 实例。
func NewSimpleMemory(cfg SimpleMemoryConfig) *SimpleMemory {
	if cfg.Dir == "" {
		cfg.Dir = "/tmp/eino/memory"
	}
	// 确保存储目录存在
	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil
	}

	return &SimpleMemory{
		dir:           cfg.Dir,
		maxWindowSize: cfg.MaxWindowSize,
		conversations: make(map[string]*Conversation),
	}
}

// SimpleMemory 实现了简单的消息存储，可以存储每个对话的消息流。
type SimpleMemory struct {
	mu            sync.Mutex
	dir           string
	maxWindowSize int
	conversations map[string]*Conversation
}

// GetConversation 根据 ID 获取对应的对话实例。
// 如果对话不存在且 createIfNotExist 为 true，则会创建一个新的。
func (m *SimpleMemory) GetConversation(id string, createIfNotExist bool) *Conversation {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.conversations[id]

	filePath := filepath.Join(m.dir, id+".jsonl")
	if !ok {
		// 如果内存中没有，检查磁盘上是否存在
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if createIfNotExist {
				// 创建新文件
				if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
					return nil
				}
				m.conversations[id] = &Conversation{
					ID:            id,
					Messages:      make([]*schema.Message, 0),
					filePath:      filePath,
					maxWindowSize: m.maxWindowSize,
				}
			}
		}

		// 磁盘上存在，则加载它
		con := &Conversation{
			ID:            id,
			Messages:      make([]*schema.Message, 0),
			filePath:      filePath,
			maxWindowSize: m.maxWindowSize,
		}
		con.load()
		m.conversations[id] = con
	}

	return m.conversations[id]
}

// ListConversations 列出所有存储的对话 ID。
func (m *SimpleMemory) ListConversations() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	files, err := os.ReadDir(m.dir)
	if err != nil {
		return nil
	}

	ids := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		// 返回去掉扩展名的文件名作为 ID
		ids = append(ids, strings.TrimSuffix(file.Name(), ".jsonl"))
	}

	return ids
}

// DeleteConversation 根据 ID 删除对话及其关联的文件。
func (m *SimpleMemory) DeleteConversation(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filePath := filepath.Join(m.dir, id+".jsonl")
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	delete(m.conversations, id)
	return nil
}

// Conversation 代表一个具体的对话及其消息历史。
type Conversation struct {
	mu sync.Mutex

	ID       string            `json:"id"`
	Messages []*schema.Message `json:"messages"`

	filePath string

	maxWindowSize int
}

// Append 向对话中追加一条新消息，并将其持久化到文件。
func (c *Conversation) Append(msg *schema.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Messages = append(c.Messages, msg)

	c.save(msg)
}

// GetFullMessages 获取该对话的所有消息。
func (c *Conversation) GetFullMessages() []*schema.Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Messages
}

// GetMessages 获取最近的 N 条消息，N 由 maxWindowSize 指定。
func (c *Conversation) GetMessages() []*schema.Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.Messages) > c.maxWindowSize {
		return c.Messages[len(c.Messages)-c.maxWindowSize:]
	}

	return c.Messages
}

// load 从磁盘文件加载消息。
func (c *Conversation) load() error {
	reader, err := os.Open(c.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		var msg schema.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}
		c.Messages = append(c.Messages, &msg)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// save 将单条消息以 JSON 基线格式追加到文件。
func (c *Conversation) save(msg *schema.Message) {
	str, _ := json.Marshal(msg)

	// 以追加模式打开文件
	f, err := os.OpenFile(c.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(str)
	f.WriteString("\n")
}
