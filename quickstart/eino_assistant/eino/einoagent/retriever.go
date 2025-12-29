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
	"fmt"
	"os"
	"strconv"

	redispkg "github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/redis"
	"github.com/cloudwego/eino/schema"
	redisCli "github.com/redis/go-redis/v9"

	"github.com/cloudwego/eino-ext/components/retriever/redis"
	"github.com/cloudwego/eino/components/retriever"
)

// newRetriever 是 EinoAgent 图中 'RedisRetriever' 节点的组件初始化函数。
// 它负责从 Redis 向量数据库中检索高度相关的文档。
func newRetriever(ctx context.Context) (rtr retriever.Retriever, err error) {
	redisAddr := os.Getenv("REDIS_ADDR")
	redisClient := redisCli.NewClient(&redisCli.Options{
		Addr:     redisAddr,
		Protocol: 2,
	})

	// 配置 Redis 检索器
	config := &redis.RetrieverConfig{
		Client:       redisClient,
		Index:        fmt.Sprintf("%s%s", redispkg.RedisPrefix, redispkg.IndexName), // 完整索引名
		Dialect:      2,                                                             // 使用 RediSearch 方言 2
		ReturnFields: []string{redispkg.ContentField, redispkg.MetadataField, redispkg.DistanceField},
		TopK:         8,                    // 返回前 8 个最相关的结果
		VectorField:  redispkg.VectorField, // 向量字段名
		// DocumentConverter 将 Redis 返回的原始数据转换为 Eino 的 Document 对象
		DocumentConverter: func(ctx context.Context, doc redisCli.Document) (*schema.Document, error) {
			resp := &schema.Document{
				ID:       doc.ID,
				Content:  "",
				MetaData: map[string]any{},
			}
			for field, val := range doc.Fields {
				if field == redispkg.ContentField {
					resp.Content = val
				} else if field == redispkg.MetadataField {
					resp.MetaData[field] = val
				} else if field == redispkg.DistanceField {
					// 将 Redis 的距离（Distance）转换为 Eino 的相关分数（1 - Distance）
					distance, err := strconv.ParseFloat(val, 64)
					if err != nil {
						continue
					}
					resp.WithScore(1 - distance)
				}
			}

			return resp, nil
		},
	}

	// 为检索器配置向量化模型（用于将查询文本向量化）
	embeddingIns11, err := newEmbedding(ctx)
	if err != nil {
		return nil, err
	}
	config.Embedding = embeddingIns11

	// 创建并返回检索器实例
	rtr, err = redis.NewRetriever(ctx, config)
	if err != nil {
		return nil, err
	}
	return rtr, nil
}
