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

// Package gitclone 提供了用于克隆或拉取 Git 仓库的工具。
package gitclone

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// GitCloneFileImpl 是 GitClone 工具的实现。
type GitCloneFileImpl struct {
	config *GitCloneFileConfig
}

// GitCloneFileConfig 包含了 GitClone 工具的配置。
type GitCloneFileConfig struct {
	BaseDir string // 仓库克隆的基础目录
}

func defaultGitCloneFileConfig(ctx context.Context) (*GitCloneFileConfig, error) {
	config := &GitCloneFileConfig{
		BaseDir: "./data/repos",
	}
	return config, nil
}

// NewGitCloneFile 创建一个新的 GitClone 工具实例。
func NewGitCloneFile(ctx context.Context, config *GitCloneFileConfig) (tn tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultGitCloneFileConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	if config.BaseDir == "" {
		return nil, fmt.Errorf("基础目录不能为空")
	}
	t := &GitCloneFileImpl{config: config}
	tn, err = t.ToEinoTool()
	if err != nil {
		return nil, err
	}
	return tn, nil
}

// ToEinoTool 将实现转换为 Eino 框架可识别的工具接口。
func (g *GitCloneFileImpl) ToEinoTool() (tool.BaseTool, error) {
	return utils.InferTool("gitclone", "克隆或拉取 GitHub 仓库", g.Invoke)
}

// Invoke 是工具执行的入口函数。
func (g *GitCloneFileImpl) Invoke(ctx context.Context, req *GitCloneRequest) (res *GitCloneResponse, err error) {
	res = &GitCloneResponse{}

	if req.Url == "" {
		res.Error = "URL 不能为空"
		return res, nil
	}

	valid, cloneURL := isValidGitURL(req.Url)
	if !valid {
		res.Error = fmt.Sprintf("无效的 Git URL 格式: %s", req.Url)
		return res, nil
	}

	repoDir, repoName := extractRepoDir(cloneURL)
	repoDir = filepath.Join(g.config.BaseDir, repoDir)
	repoPath := filepath.Join(repoDir, repoName)

	// 确保基础目录存在
	if err := os.MkdirAll(g.config.BaseDir, 0755); err != nil {
		res.Error = fmt.Sprintf("创建目录失败: %v", err)
		return res, nil
	}

	if req.Action == GitCloneActionClone {
		// 检查仓库是否已存在
		if _, err := os.Stat(repoPath); err == nil {
			res.Error = "仓库已存在"
			return res, nil
		}

		// 执行 git clone
		cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, repoPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			res.Error = fmt.Sprintf("克隆失败: %v, 输出: %s", err, output)
			return res, nil
		}
	} else if req.Action == GitCloneActionPull {
		// 检查仓库是否存在
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			res.Error = fmt.Sprintf("仓库不存在: %s", repoPath)
			return res, nil
		}

		// 执行 git pull
		cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "pull")
		if output, err := cmd.CombinedOutput(); err != nil {
			res.Error = fmt.Sprintf("拉取失败: %v, 输出: %s", err, output)
			return res, nil
		}

	}

	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		res.Error = fmt.Sprintf("获取绝对路径失败 [%s]: %v", repoPath, err)
		return res, nil
	}
	res.Message = fmt.Sprintf("成功，仓库路径为: %s", absPath)
	return res, nil
}

// 辅助函数：验证 Git URL 格式
func isValidGitURL(url string) (bool, string) {
	cleanURL := strings.TrimSuffix(url, ".git")

	parts := strings.Split(cleanURL, "/")
	if len(parts) < 2 {
		return false, ""
	}

	var standardURL string
	switch {
	// SSH 格式: git@domain:group/repo
	case strings.HasPrefix(url, "git@"):
		if strings.Contains(url, ":") {
			return true, withGit(url) // 已经是标准 SSH 格式
		}
		return false, ""

	// 完整 HTTPS 格式: https://domain/group/repo
	case strings.HasPrefix(url, "http://"), strings.HasPrefix(url, "https://"):
		return true, withGit(url) // 已经是标准 HTTPS 格式

	default:
		standardURL = "https://" + withGit(url)
	}

	return true, standardURL
}

func withGit(url string) string {
	if !strings.HasSuffix(url, ".git") {
		url += ".git"
	}
	return url
}

// 辅助函数：从 URL 提取 group 和 repo
func extractRepoDir(url string) (string, string) {
	parts := strings.Split(url, "/")
	repoDir := parts[len(parts)-2]
	repoName := strings.TrimSuffix(parts[len(parts)-1], ".git")
	return repoDir, repoName
}

// GitCloneAction 定义了工具支持的操作类型。
type GitCloneAction string

const (
	GitCloneActionClone GitCloneAction = "clone"
	GitCloneActionPull  GitCloneAction = "pull"
)

// GitCloneRequest 定义了工具的请求参数。
type GitCloneRequest struct {
	Url    string         `json:"url" jsonschema_description:"要克隆的仓库 URL"`
	Action GitCloneAction `json:"action" jsonschema_description:"要执行的操作，'clone' 或 'pull'"`
}

// GitCloneResponse 定义了工具的响应结果。
type GitCloneResponse struct {
	Message string `json:"message" jsonschema_description:"成功消息"`
	Error   string `json:"error" jsonschema_description:"错误信息"`
}
