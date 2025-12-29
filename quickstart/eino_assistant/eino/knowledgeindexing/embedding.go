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

// Package knowledgeindexing 负责知识索引流程，将文档转换为向量并存储到向量数据库。
package knowledgeindexing

import (
	"context"
	"os"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino/components/embedding"
)

// newEmbedding 是 KnowledgeIndexing 图中使用的向量化组件初始化函数。
// 它对接了火山引擎 Ark 的向量模型。
func newEmbedding(ctx context.Context) (eb embedding.Embedder, err error) {
	// 配置 Ark 向量化组件
	config := &ark.EmbeddingConfig{
		BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
		APIKey:  os.Getenv("ARK_API_KEY"),         // 从环境变量获取 API Key
		Model:   os.Getenv("ARK_EMBEDDING_MODEL"), // 从环境变量获取模型 ID
	}
	eb, err = ark.NewEmbedder(ctx, config)
	if err != nil {
		return nil, err
	}
	return eb, nil
}
