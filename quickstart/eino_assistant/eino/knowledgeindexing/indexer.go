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

package knowledgeindexing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/indexer/redis"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	redisCli "github.com/redis/go-redis/v9"

	redispkg "github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/redis"
)

func init() {
	// 初始化 Redis 向量索引环境（创建索引 Schema 等）
	err := redispkg.Init()
	if err != nil {
		log.Fatalf("初始化 Redis 索引失败: %v", err)
	}
}

// newIndexer 是 KnowledgeIndexing 图中 'RedisIndexer' 节点的组件初始化函数。
// 它负责将处理后的文档块及生成的向量存储到 Redis 中。
func newIndexer(ctx context.Context) (idr indexer.Indexer, err error) {
	redisAddr := os.Getenv("REDIS_ADDR")
	redisClient := redisCli.NewClient(&redisCli.Options{
		Addr:     redisAddr,
		Protocol: 2, // 使用 RESP2 协议
	})

	config := &redis.IndexerConfig{
		Client:    redisClient,
		KeyPrefix: redispkg.RedisPrefix, // Redis 键前缀
		BatchSize: 1,                    // 批处理大小
		// DocumentToHashes 定义了如何将 Eino 的 Document 对象映射为 Redis 的 Hash 结构
		DocumentToHashes: func(ctx context.Context, doc *schema.Document) (*redis.Hashes, error) {
			if doc.ID == "" {
				doc.ID = uuid.New().String()
			}
			key := doc.ID

			// 将元数据序列化为 JSON 字符串
			metadataBytes, err := json.Marshal(doc.MetaData)
			if err != nil {
				return nil, fmt.Errorf("序列化元数据失败: %w", err)
			}

			return &redis.Hashes{
				Key: key,
				Field2Value: map[string]redis.FieldValue{
					// 内容字段：存储原始文本，且指定对应的向量存储字段
					redispkg.ContentField:  {Value: doc.Content, EmbedKey: redispkg.VectorField},
					redispkg.MetadataField: {Value: metadataBytes},
				},
			}, nil
		},
	}

	// 初始化用于生成向量的 Embedding 组件
	embeddingIns11, err := newEmbedding(ctx)
	if err != nil {
		return nil, err
	}
	config.Embedding = embeddingIns11

	// 创建并返回 Redis Indexer 实例
	idr, err = redis.NewIndexer(ctx, config)
	if err != nil {
		return nil, err
	}
	return idr, nil
}
