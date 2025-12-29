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

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/compose"
)

// BuildKnowledgeIndexing 构建并编译知识索引的任务流图。
// 流程：开始 -> 加载文件 -> Markdown 拆分 -> Redis 索引 -> 结束。
func BuildKnowledgeIndexing(ctx context.Context) (r compose.Runnable[document.Source, []string], err error) {
	// 定义节点 Key
	const (
		FileLoader       = "FileLoader"       // 文件加载节点
		MarkdownSplitter = "MarkdownSplitter" // Markdown 拆分节点
		RedisIndexer     = "RedisIndexer"     // Redis 索引存储节点
	)

	// 创建一个新的图，输入为文件源信息，输出为存储后的文档 ID 列表
	g := compose.NewGraph[document.Source, []string]()

	// 1. 添加加载器节点
	fileLoaderKeyOfLoader, err := newLoader(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddLoaderNode(FileLoader, fileLoaderKeyOfLoader)

	// 2. 添加处理（拆分）节点
	markdownSplitterKeyOfDocumentTransformer, err := newDocumentTransformer(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddDocumentTransformerNode(MarkdownSplitter, markdownSplitterKeyOfDocumentTransformer)

	// 3. 添加索引存储节点
	redisIndexerKeyOfIndexer, err := newIndexer(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddIndexerNode(RedisIndexer, redisIndexerKeyOfIndexer)

	// 4. 连接节点编排流程
	_ = g.AddEdge(compose.START, FileLoader)
	_ = g.AddEdge(FileLoader, MarkdownSplitter)
	_ = g.AddEdge(MarkdownSplitter, RedisIndexer)
	_ = g.AddEdge(RedisIndexer, compose.END)

	// 5. 编译图，生成的 Runnable 可用于执行具体任务
	r, err = g.Compile(ctx, compose.WithGraphName("KnowledgeIndexing"), compose.WithNodeTriggerMode(compose.AnyPredecessor))
	if err != nil {
		return nil, err
	}
	return r, err
}
