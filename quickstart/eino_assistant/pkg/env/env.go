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

// Package env 提供环境变量管理功能，支持从 .env 文件加载配置。
package env

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	// 加载项目根目录下的 .env 文件
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("❌ [错误] 无法加载 .env 文件: %v", err)
	}

}

// MustHasEnvs 检查指定的环境变量是否存在。
// 如果任何一个环境变量未设置，程序将打印错误并退出。
func MustHasEnvs(envs ...string) {
	for _, env := range envs {
		if os.Getenv(env) == "" {
			log.Fatalf("❌ [错误] 环境变量 [%s] 是必需的，但当前未设置，请检查您的 .env 文件", env)
		}
	}
}
