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

// Package main 是 Eino 助手 Web 服务的入口。
// 它基于 Hertz 框架启动一个 HTTP 服务器，提供任务管理和智能对话的 Web 接口。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/devops"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/obs-opentelemetry/provider"
	hertztracing "github.com/hertz-contrib/obs-opentelemetry/tracing"
	"go.opentelemetry.io/otel/attribute"

	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/cmd/einoagent/agent"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/cmd/einoagent/task"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/env"
)

func init() {
	// 如果不是生产环境，则开启 Eino 可视化调试工具 (DevOps)
	if os.Getenv("EINO_DEBUG") != "false" {
		err := devops.Init(context.Background())
		if err != nil {
			log.Printf("[Eino 调试] 初始化失败, 错误=%v", err)
		}
	}

	// 检查运行所需的关键环境变量
	env.MustHasEnvs("CHAT_MODEL", "EMBEDDING_MODEL", "CHAT_MODEL_BASE_URL", "EMBEDDING_MODEL_BASE_URL")
}

func main() {
	// 设置监听端口，默认 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 初始化 Hertz HTTP 服务器
	h := server.Default(server.WithHostPorts(":" + port))

	// 使用简易日志中间件
	h.Use(LogMiddleware())

	// 如果配置了 APMPlus，则集成 OpenTelemetry 链路追踪
	if os.Getenv("APMPLUS_APP_KEY") != "" {
		region := os.Getenv("APMPLUS_REGION")
		if region == "" {
			region = "cn-beijing"
		}
		_ = provider.NewOpenTelemetryProvider(
			provider.WithServiceName("eino-assistant"),
			provider.WithExportEndpoint(fmt.Sprintf("apmplus-%s.volces.com:4317", region)),
			provider.WithInsecure(),
			provider.WithHeaders(map[string]string{"X-ByteAPM-AppKey": os.Getenv("APMPLUS_APP_KEY")}),
			provider.WithResourceAttribute(attribute.String("apmplus.business_type", "llm")),
		)
		tracer, cfg := hertztracing.NewServerTracer()
		h = server.Default(server.WithHostPorts(":"+port), tracer)
		h.Use(LogMiddleware(), hertztracing.ServerMiddleware(cfg))
	}

	// 1. 注册任务管理模块路由 (URL 前缀: /task)
	taskGroup := h.Group("/task")
	if err := task.BindRoutes(taskGroup); err != nil {
		log.Fatal("绑定任务模块路由失败:", err)
	}

	// 2. 注册智能体对话模块路由 (URL 前缀: /agent)
	agentGroup := h.Group("/agent")
	if err := agent.BindRoutes(agentGroup); err != nil {
		log.Fatal("绑定对话模块路由失败:", err)
	}

	// 将根路径重定向到对话页面
	h.GET("/", func(ctx context.Context, c *app.RequestContext) {
		c.Redirect(302, []byte("/agent"))
	})

	// 启动服务器并挂起监听
	h.Spin()
}

// LogMiddleware 用于在控制台输出 HTTP 请求的基础统计信息
func LogMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		path := string(c.Request.URI().Path())
		method := string(c.Request.Method())

		// 处理下游逻辑
		c.Next(ctx)

		// 计算并记录响应时间
		latency := time.Since(start)
		statusCode := c.Response.StatusCode()
		log.Printf("[HTTP 请求] %s %s %d 处理耗时 %v\n", method, path, statusCode, latency)
	}
}
