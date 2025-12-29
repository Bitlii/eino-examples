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

// Package redis 提供了基于 Redis Stack 的向量索引初始化和管理功能。
package redis

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

const (
	// RedisPrefix 是文档和索引的前缀
	RedisPrefix = "eino:doc:"
	// IndexName 是向量索引的名称
	IndexName = "vector_index"

	// 字段名称定义
	ContentField  = "content"        // 内容字段
	MetadataField = "metadata"       // 元数据字段
	VectorField   = "content_vector" // 向量字段
	DistanceField = "distance"       // 距离字段（查询时返回）
)

var initOnce sync.Once

// Init 提供了一个简单的入口来初始化 Redis 索引。
// 它会按默认设置（本地地址和 4096 维度）执行初始化，且只执行一次。
func Init() error {
	var err error
	initOnce.Do(func() {
		err = InitRedisIndex(context.Background(), &Config{
			RedisAddr: "localhost:6379",
			Dimension: 4096,
		})
	})
	return err
}

// Config 包含了 Redis 连接和索引配置。
type Config struct {
	RedisAddr string // Redis 服务器地址
	Dimension int    // 向量维度
}

// InitRedisIndex 在 Redis 中创建并初始化向量索引。
// 如果索引已存在，则直接返回。
func InitRedisIndex(ctx context.Context, config *Config) (err error) {
	if config.Dimension <= 0 {
		return fmt.Errorf("维度必须为正数")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Protocol: 2, // 使用 RESP2 协议
	})

	// 确保在出错时关闭客户端，正常情况下可能需要保持长连接
	defer func() {
		if err != nil {
			client.Close()
		}
	}()

	// 检查 Redis 连接
	if err = client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("无法连接到 Redis: %w", err)
	}

	indexName := fmt.Sprintf("%s%s", RedisPrefix, IndexName)

	// 检查是否存在同名索引
	exists, err := client.Do(ctx, "FT.INFO", indexName).Result()
	if err != nil {
		// 如果错误包含 "Unknown index name"，说明索引不存在，可以创建
		if !strings.Contains(err.Error(), "Unknown index name") {
			return fmt.Errorf("检查索引是否存在失败: %w", err)
		}
		err = nil
	} else if exists != nil {
		// 索引已存在
		return nil
	}

	// 创建新索引
	// 使用命令: FT.CREATE <index_name> ON HASH PREFIX 1 <prefix> SCHEMA <field1> TEXT <field2> TEXT <vector_field> VECTOR FLAT ...
	createIndexArgs := []interface{}{
		"FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", RedisPrefix,
		"SCHEMA",
		ContentField, "TEXT",
		MetadataField, "TEXT",
		VectorField, "VECTOR", "FLAT",
		"6", // 参数数量
		"TYPE", "FLOAT32",
		"DIM", config.Dimension,
		"DISTANCE_METRIC", "COSINE", // 距离度量使用余弦相似度
	}

	if err = client.Do(ctx, createIndexArgs...).Err(); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	// 验证索引是否创建成功
	if _, err = client.Do(ctx, "FT.INFO", indexName).Result(); err != nil {
		return fmt.Errorf("验证索引创建失败: %w", err)
	}

	return nil
}
