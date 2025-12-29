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

// Package open 提供了在系统中通过默认应用程序打开文件、目录或 Web URL 的工具。
package open

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// OpenFileToolImpl 是 Open 工具的实现。
type OpenFileToolImpl struct {
	config *OpenFileToolConfig
}

// OpenFileToolConfig 包含了 Open 工具的配置。
type OpenFileToolConfig struct {
}

func defaultOpenFileToolConfig(ctx context.Context) (*OpenFileToolConfig, error) {
	config := &OpenFileToolConfig{}
	return config, nil
}

// NewOpenFileTool 创建一个新的 Open 工具实例。
func NewOpenFileTool(ctx context.Context, config *OpenFileToolConfig) (tn tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultOpenFileToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	t := &OpenFileToolImpl{config: config}
	tn, err = t.ToEinoTool()
	if err != nil {
		return nil, err
	}
	return tn, nil
}

// ToEinoTool 将实现转换为 Eino 框架可识别的工具接口。
func (of *OpenFileToolImpl) ToEinoTool() (tool.InvokableTool, error) {
	return utils.InferTool("open", "通过系统默认应用程序打开文件、目录或 Web URL", of.Invoke)
}

// Invoke 是工具执行的入口函数。
func (of *OpenFileToolImpl) Invoke(ctx context.Context, req OpenReq) (res OpenRes, err error) {
	if req.URI == "" {
		res.Message = "URI 是必需的"
		return res, nil
	}

	// 如果是本地文件或目录，检查其是否存在
	if isFilePath(req.URI) {
		req.URI = strings.TrimPrefix(req.URI, "file:///")
		if _, err := os.Stat(req.URI); err != nil {
			res.Message = fmt.Sprintf("文件不存在: %s", req.URI)
			return res, nil
		}
	}

	// 调用系统命令打开 URI
	err = openURI(req.URI)
	if err != nil {
		res.Message = fmt.Sprintf("无法打开 %s: %s", req.URI, err.Error())
		return res, nil
	}

	res.Message = fmt.Sprintf("成功打开 %s", req.URI)
	return res, nil
}

// OpenReq 定义了工具的请求参数。
type OpenReq struct {
	URI string `json:"uri" jsonschema_description:"要打开的文件、目录或 Web URL 的 URI"`
}

// OpenRes 定义了工具的响应结果。
type OpenRes struct {
	Message string `json:"message" jsonschema_description:"操作结果消息"`
}

// openURI 根据操作系统的不同调用相应的命令。
func openURI(uri string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", uri)
	case "darwin":
		cmd = exec.Command("open", uri)
	case "linux":
		cmd = exec.Command("xdg-open", uri)
	default:
		return fmt.Errorf("不支持的平台")
	}
	return cmd.Run()
}

// isFilePath 判断给定的 URI 是否代表一个本地文件路径。
func isFilePath(path string) bool {
	s, err := url.Parse(path)
	return err == nil && s.Scheme == "file" && s.Path != ""
}
