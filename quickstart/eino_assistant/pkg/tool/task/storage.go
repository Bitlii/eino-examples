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

// Package task 提供了任务管理功能，支持任务的增删改查及持久化存储。
package task

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var defaultStorage *Storage

// Storage 负责任务的持久化存储和内存缓存管理。
type Storage struct {
	filePath string           // 数据文件路径 (.jsonl 格式)
	mu       sync.RWMutex     // 保护并发读写的锁
	cache    map[string]*Task // 内存缓存，ID 到 Task 的映射
	dirty    bool             // 标记是否有未同步到磁盘的变更
}

// GetDefaultStorage 返回默认的 Storage 实例，如果未初始化则先初始化。
func GetDefaultStorage() *Storage {
	if defaultStorage == nil {
		InitDefaultStorage("./data/task")
	}
	return defaultStorage
}

// InitDefaultStorage 初始化默认的 Storage 实例。
func InitDefaultStorage(dataDir string) error {
	s, err := NewStorage(dataDir)
	if err != nil {
		return err
	}
	defaultStorage = s
	return nil
}

// NewStorage 创建一个新的 Storage 实例，并从磁盘加载已有任务。
func NewStorage(dataDir string) (*Storage, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("无法创建数据目录: %v", err)
	}
	s := &Storage{
		filePath: filepath.Join(dataDir, "tasks.jsonl"),
		cache:    make(map[string]*Task),
	}

	// 启动时从磁盘加载数据
	if err := s.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("从磁盘加载数据失败: %v", err)
	}

	return s, nil
}

// loadFromDisk 从 JSONL 文件读取任务并填充到内存缓存。
func (s *Storage) loadFromDisk() error {
	file, err := os.OpenFile(s.filePath, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法打开文件: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var task Task
		if err := json.Unmarshal(scanner.Bytes(), &task); err != nil {
			return fmt.Errorf("解析任务失败: %v", err)
		}
		s.cache[task.ID] = &task
	}

	return scanner.Err()
}

// Add 向存储中添加一个新任务，并立即追加到文件末尾。
func (s *Storage) Add(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task.CreatedAt = time.Now().Format(time.RFC3339)
	task.IsDeleted = false
	s.cache[task.ID] = task

	// 直接以追加模式写入文件，避免重写整个文件
	file, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法打开文件: %v", err)
	}
	defer file.Close()

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("序列化任务失败: %v", err)
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("写入任务失败: %v", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("同步文件到磁盘失败: %v", err)
	}

	return nil
}

// List 根据参数过滤并返回任务列表。
// 默认排序规则：未完成任务在前，已完成在后；组内按创建时间降序排列。
func (s *Storage) List(params *ListParams) ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var activeTasks, completedTasks []*Task
	for _, task := range s.cache {
		if task.IsDeleted {
			continue
		}

		// 文本查询过滤（标题或内容中包含关键字）
		if params.Query != "" && !contains(task.Title, params.Query) && !contains(task.Content, params.Query) {
			continue
		}

		// 完成状态过滤
		if params.IsDone != nil {
			if task.Completed != *params.IsDone {
				continue
			}
		}

		if task.Completed {
			completedTasks = append(completedTasks, task)
		} else {
			activeTasks = append(activeTasks, task)
		}
	}

	// 排序逻辑：最新的在前面
	sort.Slice(activeTasks, func(i, j int) bool {
		return activeTasks[i].CreatedAt > activeTasks[j].CreatedAt
	})
	sort.Slice(completedTasks, func(i, j int) bool {
		return completedTasks[i].CreatedAt > completedTasks[j].CreatedAt
	})

	// 合并列表：未完成的在前
	tasks := append(activeTasks, completedTasks...)

	// 分页限额
	if params.Limit != nil && len(tasks) > *params.Limit {
		tasks = tasks[:*params.Limit]
	}

	return tasks, nil
}

// Update 更新现有任务的字段内容。
func (s *Storage) Update(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.cache[task.ID]
	if !exists || existing.IsDeleted {
		return fmt.Errorf("未找到任务: %s", task.ID)
	}

	// 只更新传入任务中非空的字段（模拟局部更新）
	updated := *existing
	if task.Title != "" {
		updated.Title = task.Title
	}
	if task.Content != "" {
		updated.Content = task.Content
	}
	if task.Deadline != "" {
		updated.Deadline = task.Deadline
	}
	// Completed 字段始终根据传入值更新
	if task.Completed != existing.Completed {
		updated.Completed = task.Completed
	}

	s.cache[task.ID] = &updated
	s.dirty = true

	return s.syncToDisk()
}

// Delete 标记删除一个任务。
func (s *Storage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.cache[id]
	if !exists || task.IsDeleted {
		return fmt.Errorf("未找到任务: %s", id)
	}

	// 这里采用软删除模式
	task.IsDeleted = true
	s.dirty = true

	return s.syncToDisk()
}

// syncToDisk 将内存中的所有任务全量同步到磁盘文件。
// 为了保证安全性，使用了先写临时文件再重命名的原子操作模式。
func (s *Storage) syncToDisk() error {
	if !s.dirty {
		return nil
	}

	// 1. 创建并写入临时文件
	tmpFile := s.filePath + ".tmp"
	file, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("无法创建临时文件: %v", err)
	}
	defer file.Close()

	for _, task := range s.cache {
		data, err := json.Marshal(task)
		if err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("序列化任务失败: %v", err)
		}

		if _, err := file.Write(append(data, '\n')); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("写入任务失败: %v", err)
		}
	}

	// 2. 确保数据刷入物理磁盘
	if err := file.Sync(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("同步文件失败: %v", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("关闭文件失败: %v", err)
	}

	// 3. 备份旧文件并替换为新文件
	if _, err := os.Stat(s.filePath); err == nil {
		// 备份现有数据以防万一
		backupFile := s.filePath + ".bak"
		if err := os.Rename(s.filePath, backupFile); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("无法创建备份文件: %v", err)
		}
	}

	// 重命名临时文件为正式文件（原子替换）
	if err := os.Rename(tmpFile, s.filePath); err != nil {
		// 如果重命名失败，尝试回滚备份
		if backupErr := os.Rename(s.filePath+".bak", s.filePath); backupErr != nil {
			return fmt.Errorf("临时文件重命名失败，且无法恢复备份: %v, 备份错误: %v", err, backupErr)
		}
		return fmt.Errorf("重命名临时文件失败: %v", err)
	}

	// 4. 清理备份并清除脏标记
	os.Remove(s.filePath + ".bak")

	s.dirty = false
	return nil
}

// contains 助手函数，实现不区分大小写的字符串包含检查。
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
