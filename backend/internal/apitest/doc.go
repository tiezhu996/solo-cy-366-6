// Package apitest 外设租借与损坏赔付闭环的 HTTP 级集成测试。
// 使用内存 SQLite 装配真实 Gin 引擎与全部业务分层（handler→service→repository），
// 每个测试独立建库、自建数据、结束即销毁，可重复运行且结果稳定。
package apitest
