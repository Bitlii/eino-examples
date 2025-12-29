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

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino/components/document"
)

// newDocumentTransformer 是 KnowledgeIndexing 图中 'MarkdownSplitter' 节点的组件初始化函数。
// 它负责将加载的 Markdown 文档按照标题层级进行拆分。
func newDocumentTransformer(ctx context.Context) (tfr document.Transformer, err error) {
	// 配置 Markdown 标题拆分器
	config := &markdown.HeaderConfig{
		Headers: map[string]string{
			"#": "title", // 将一级标题识别为标题元数据
		},
		TrimHeaders: false}
	tfr, err = markdown.NewHeaderSplitter(ctx, config)
	if err != nil {
		return nil, err
	}
	return tfr, nil
}
