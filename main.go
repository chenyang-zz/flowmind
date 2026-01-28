/**
 * FlowMind 主入口文件
 *
 * 这是 Wails 应用的启动点，负责：
 * 1. 初始化应用配置
 * 2. 创建 App 实例
 * 3. 启动 Wails 运行时
 */

package main

import (
	"context"
	"embed"
	"log"

	"github.com/chenyang-zz/flowmind/internal/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

/**
 * 主函数
 *
 * 应用的入口点，负责初始化并启动 Wails 应用
 */
func main() {
	// 创建 App 实例
	// App 是前后端通信的桥梁，包含所有导出的方法
	flowmindApp := app.New()

	// 启动 Wails 应用
	err := wails.Run(&options.App{
		// ========== 应用基本配置 ==========

		/** 应用标题 */
		Title: "FlowMind",

		/** 应用窗口宽度（像素） */
		Width: 1280,

		/** 应用窗口高度（像素） */
		Height: 800,

		/** 窗口背景色 (白色) */
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 255},

		// ========== Asset Server 配置 ==========

		/**
		 * AssetServer 配置
		 * 用于服务前端静态文件
		 */
		AssetServer: &assetserver.Options{
			Assets: assets,
		},

		// ========== 绑定 App 实例 ==========

		/**
		 * 将 App 实例绑定到 Wails
		 * 前端可以通过 window.runtime.go/main/App 访问导出的方法
		 */
		Bind: []interface{}{
			flowmindApp,
		},

		// ========== 启动时回调 ==========

		/**
		 * OnStartup 在应用启动时调用
		 * 用于初始化应用状态、启动后台服务等
		 */
		OnStartup: func(ctx context.Context) {
			// 初始化应用
			if err := flowmindApp.Startup(ctx); err != nil {
				log.Fatalf("Failed to startup: %v", err)
			}
		},

		/**
		 * OnDomReady 在前端 DOM 准备好时调用
		 * 此时可以安全地调用前端方法
		 */
		OnDomReady: func(ctx context.Context) {
			// DOM 已准备好，可以进行前端交互
		},

		/**
		 * OnShutdown 在应用关闭时调用
		 * 用于清理资源、保存状态等
		 */
		OnShutdown: func(ctx context.Context) {
			// 清理资源
			flowmindApp.Shutdown()
		},

		/**
		 * OnBeforeClose 在应用关闭前调用
		 * 可以返回 false 阻止关闭
		 */
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			// 可以在这里提示用户保存未保存的工作
			// return true // 取消关闭
			return false // 允许关闭
		},
	})

	// 如果启动失败，记录错误并退出
	if err != nil {
		log.Fatalf("Error: %s", err.Error())
	}
}
