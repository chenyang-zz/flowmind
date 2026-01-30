// Package services 提供业务流程编排
//
// Service 层负责协调多个领域模块，实现应用级用例。
// 它不包含核心业务逻辑（在 Domain 层），而是编排和协调。
//
// 职责：
//   - 编排多个 Domain 协作
//   - 实现应用级用例
//   - 处理事务和错误
//   - 与 App 层交互

package services
