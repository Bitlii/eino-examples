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

// Package main 是知识索引工具的入口。
// 它遍历本地 Markdown 文件，通过 Eino 工作流将其向量化并存储到 Redis。
package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/env"
	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/coze-dev/cozeloop-go"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/redis/go-redis/v9"

	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/eino/knowledgeindexing"
)

func init() {
	// 检查运行所需的关键环境变量
	env.MustHasEnvs("EMBEDDING_MODEL_BASE_URL", "EMBEDDING_MODEL")
}

func main() {
	// 获取可选的 CozeLoop 可视化追踪 Token
	cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID") // 更多信息见 https://loop.coze.cn/open/docs/cozeloop/go-sdk

	ctx := context.Background()
	var handlers []callbacks.Handler
	// 如果配置了 CozeLoop，则添加对应的全局回调处理器
	if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
		client, err := cozeloop.NewClient(
			cozeloop.WithAPIToken(cozeloopApiToken),
			cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
		)
		if err != nil {
			panic(err)
		}
		defer client.Close(ctx)
		handlers = append(handlers, clc.NewLoopHandler(client))
	}
	callbacks.AppendGlobalHandlers(handlers...)

	// 对 ./eino-docs 目录下的所有 Markdown 文件进行 RAG 索引
	err := indexMarkdownFiles(ctx, "./eino-docs")
	if err != nil {
		panic(err)
	}

	fmt.Println("索引构建成功！")
}

// indexMarkdownFiles 遍历指定目录并逐个处理 Markdown 文件
func indexMarkdownFiles(ctx context.Context, dir string) error {
	// 构建知识索引的工作流图
	runner, err := knowledgeindexing.BuildKnowledgeIndexing(ctx)
	if err != nil {
		return fmt.Errorf("构建索引图失败: %w", err)
	}

	// 遍历并处理
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("遍历目录失败: %w", err)
		}
		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".md") {
			fmt.Printf("[跳过] 非 Markdown 文件: %s\n", path)
			return nil
		}

		fmt.Printf("[执行] 正在索引文件: %s\n", path)

		// 调用工作流进行：加载 -> 拆分 -> 向量化 -> 存储
		ids, err := runner.Invoke(ctx, document.Source{URI: path})
		if err != nil {
			return fmt.Errorf("执行索引任务失败: %w", err)
		}

		fmt.Printf("[完成] 索引文件: %s, 拆分块数: %d\n", path, len(ids))

		return nil
	})

	return err
}

// RedisVectorStoreConfig 定义了 Redis 向量库的配置
type RedisVectorStoreConfig struct {
	RedisKeyPrefix string             // Redis 键前缀
	IndexName      string             // 索引名称
	Embedding      embedding.Embedder // 向量生成器
	Dimension      int                // 向量维度
	MinScore       float64            // 相关性最小分数
	RedisAddr      string             // Redis 服务器地址
}

// initVectorIndex 显式初始化 Redis 向量索引（如果尚未创建）
func initVectorIndex(ctx context.Context, config *RedisVectorStoreConfig) (err error) {
	if config.Embedding == nil {
		return fmt.Errorf("向量化组件不能为空")
	}
	if config.Dimension <= 0 {
		return fmt.Errorf("向量维度必须为正数")
	}

	client := redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
	})

	defer func() {
		if err != nil {
			client.Close()
		}
	}()

	if err = client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("连接 Redis 失败: %w", err)
	}

	indexName := fmt.Sprintf("%s%s", config.RedisKeyPrefix, config.IndexName)

	// 检查索引是否已存在
	exists, err := client.Do(ctx, "FT.INFO", indexName).Result()
	if err != nil {
		if !strings.Contains(err.Error(), "Unknown index name") {
			return fmt.Errorf("检查索引是否存在失败: %w", err)
		}
		err = nil
	} else if exists != nil {
		return nil
	}

	// 执行创建索引的命令：FT.CREATE ...
	createIndexArgs := []interface{}{
		"FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", config.RedisKeyPrefix,
		"SCHEMA",
		"content", "TEXT",
		"metadata", "TEXT",
		"vector", "VECTOR", "FLAT",
		"6",
		"TYPE", "FLOAT32",
		"DIM", config.Dimension,
		"DISTANCE_METRIC", "COSINE",
	}

	if err = client.Do(ctx, createIndexArgs...).Err(); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	// 再次验证索引状态
	if _, err = client.Do(ctx, "FT.INFO", indexName).Result(); err != nil {
		return fmt.Errorf("验证索引创建失败: %w", err)
	}

	return nil
}
