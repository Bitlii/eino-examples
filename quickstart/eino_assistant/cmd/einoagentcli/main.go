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

// Package main 是 Eino 助手的命令行交互终端 (CLI)。
// 它允许用户在终端中以会话形式与 Eino 助手进行实时对话，并自动管理对话历史。
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/cloudwego/eino-ext/callbacks/apmplus"
	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino-ext/callbacks/langfuse"
	"github.com/cloudwego/eino-ext/devops"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"

	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/eino/einoagent"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/env"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/mem"
)

// 定义命令行参数
var id = flag.String("id", "", "会话 ID，如果不提供则随机生成")

var memory = mem.GetDefaultMemory() // 会话持久化存储
var cbHandler callbacks.Handler     // 运行时回调处理器

func main() {
	flag.Parse()

	// 开启 Eino DevOps 可视化监控
	err := devops.Init(context.Background())
	if err != nil {
		log.Printf("[Eino 调试] 初始化失败, 错误=%v", err)
		return
	}

	// 如果未指定会话 ID，则生成一个随机数字作为 ID
	if *id == "" {
		*id = strconv.Itoa(rand.Intn(1000000))
	}

	ctx := context.Background()

	// 初始化环境及各种追踪/指标回调
	err = Init()
	if err != nil {
		log.Printf("[助手机器人] 初始化失败, 错误=%v", err)
		return
	}

	fmt.Printf("机器人会话 ID: %s (输入 'exit' 或 'quit' 退出)\n", *id)

	// 进入交互式对话循环
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("🧑‍ : ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("读取输入错误: %v\n", err)
			return
		}

		input = strings.TrimSpace(input)
		// 检查退出指令
		if input == "" || input == "exit" || input == "quit" {
			return
		}

		// 调用核心流程处理用户输入
		sr, err := RunAgent(ctx, *id, input)
		if err != nil {
			fmt.Printf("运行机器人出错: %v\n", err)
			continue
		}

		// 打印机器人回复头
		fmt.Print("🤖 : ")
		for {
			// 从流中逐字接收回复内容
			msg, err := sr.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Printf("接收消息流异常: %v\n", err)
				break
			}
			fmt.Print(msg.Content)
		}
		fmt.Println()
		fmt.Println()
	}
}

// Init 负责加载配置并初始化各种外围链路（监控、日志等）
func Init() error {
	// 校验必要环境变量
	env.MustHasEnvs("ARK_CHAT_MODEL", "ARK_EMBEDDING_MODEL", "ARK_API_KEY")

	os.MkdirAll("log", 0755)
	// 开启本地运行日志
	f, err := os.OpenFile("log/eino.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	cbConfig := &LogCallbackConfig{
		Detail: true,
		Writer: f,
	}
	if os.Getenv("DEBUG") == "true" {
		cbConfig.Debug = true
	}
	// 创建本地日志回调
	cbHandler = LogCallback(cbConfig)

	// 组装各类第三方追踪系统的回调处理器
	callbackHandlers := make([]callbacks.Handler, 0)

	// 1. 集成字节跳动 APMPlus
	if os.Getenv("APMPLUS_APP_KEY") != "" {
		region := os.Getenv("APMPLUS_REGION")
		if region == "" {
			region = "cn-beijing"
		}
		fmt.Println("[集成消息] 正在使用 APMPlus 进行调用链追踪")
		cbh, _, err := apmplus.NewApmplusHandler(&apmplus.Config{
			Host:        fmt.Sprintf("apmplus-%s.volces.com:4317", region),
			AppKey:      os.Getenv("APMPLUS_APP_KEY"),
			ServiceName: "eino-assistant-cli",
			Release:     "release/v0.0.1",
		})
		if err != nil {
			log.Fatal(err)
		}
		callbackHandlers = append(callbackHandlers, cbh)
	}

	// 2. 集成全链路追踪平台 Langfuse
	if os.Getenv("LANGFUSE_PUBLIC_KEY") != "" && os.Getenv("LANGFUSE_SECRET_KEY") != "" {
		fmt.Println("[集成消息] 正在使用 Langfuse 进行 LLM 任务追踪")
		cbh, _ := langfuse.NewLangfuseHandler(&langfuse.Config{
			Host:      "https://cloud.langfuse.com",
			PublicKey: os.Getenv("LANGFUSE_PUBLIC_KEY"),
			SecretKey: os.Getenv("LANGFUSE_SECRET_KEY"),
			Name:      "Eino Assistant CLI",
			Public:    true,
			Release:   "release/v0.0.1",
			UserID:    "eino_cli_user",
			Tags:      []string{"cli", "interactive"},
		})
		callbackHandlers = append(callbackHandlers, cbh)
	}

	// 3. 集成 CozeLoop 可视化工具
	cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")
	if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
		client, err := cozeloop.NewClient(
			cozeloop.WithAPIToken(cozeloopApiToken),
			cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
		)
		if err != nil {
			panic(err)
		}
		// 延迟关闭客户端需要谨慎，此处逻辑仅作为初始化参考
		callbackHandlers = append(callbackHandlers, clc.NewLoopHandler(client))
	}

	// 初始化 Eino 全局回调
	if len(callbackHandlers) > 0 {
		callbacks.InitCallbackHandlers(callbackHandlers)
	}

	return nil
}

// RunAgent 负责执行一次 AI 生成任务，并处理历史对话的存储逻辑
func RunAgent(ctx context.Context, id string, msg string) (*schema.StreamReader[*schema.Message], error) {
	// 构建智能体执行图
	runner, err := einoagent.BuildEinoAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("构建智能体图失败: %w", err)
	}

	// 获取并加载历史对话
	conversation := memory.GetConversation(id, true)

	userMessage := &einoagent.UserMessage{
		ID:      id,
		Query:   msg,
		History: conversation.GetMessages(),
	}

	// 启动流式调用，并挂载运行时回调处理器
	sr, err := runner.Stream(ctx, userMessage, compose.WithCallbacks(cbHandler))
	if err != nil {
		return nil, fmt.Errorf("流式调用异常: %w", err)
	}

	// 复制一份流，用于异步将回复内容持久化到历史记录中
	srs := sr.Copy(2)

	go func() {
		// 用于收集完整的 AI 回复
		fullMsgs := make([]*schema.Message, 0)

		defer func() {
			srs[1].Close()
			// 将用户本次输入追加到历史记录
			conversation.Append(schema.UserMessage(msg))

			// 合并流片段为完整消息并存入历史
			fullMsg, err := schema.ConcatMessages(fullMsgs)
			if err != nil {
				fmt.Println("拼接回复消息失败: ", err.Error())
				return
			}
			conversation.Append(fullMsg)
		}()

	outer:
		for {
			select {
			case <-ctx.Done():
				return
			default:
				chunk, err := srs[1].Recv()
				if err != nil {
					if errors.Is(err, io.EOF) {
						break outer
					}
				}
				fullMsgs = append(fullMsgs, chunk)
			}
		}
	}()

	return srs[0], nil
}

// LogCallbackConfig 日志回调配置
type LogCallbackConfig struct {
	Detail bool      // 是否记录详细的 Input/Output
	Debug  bool      // 是否使用格式化的 Indent 输出
	Writer io.Writer // 日志输出目标
}

// LogCallback 创建一个简单的控制台/文件日志回调处理器，便于在终端观察 Eino 节点的内部流转
func LogCallback(config *LogCallbackConfig) callbacks.Handler {
	if config == nil {
		config = &LogCallbackConfig{
			Detail: true,
			Writer: os.Stdout,
		}
	}
	if config.Writer == nil {
		config.Writer = os.Stdout
	}
	builder := callbacks.NewHandlerBuilder()
	// 当节点开始执行时触发
	builder.OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
		fmt.Fprintf(config.Writer, "[流程观测]: 开始节点 [%s:%s:%s]\n", info.Component, info.Type, info.Name)
		if config.Detail {
			var b []byte
			if config.Debug {
				b, _ = json.MarshalIndent(input, "", "  ")
			} else {
				b, _ = json.Marshal(input)
			}
			fmt.Fprintf(config.Writer, "输入数据: %s\n", string(b))
		}
		return ctx
	})
	// 当节点执行结束时触发
	builder.OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
		fmt.Fprintf(config.Writer, "[流程观测]: 结束节点 [%s:%s:%s]\n", info.Component, info.Type, info.Name)
		return ctx
	})
	return builder.Build()
}
