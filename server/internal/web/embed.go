package web

import (
	"embed"
	"io/fs"
)

//go:embed dist dist/*
// embedded 是一个嵌入式文件系统，包含编译到二进制中的Web前端资源
// 通过 go:embed 指令将 dist 目录下的所有文件嵌入到二进制中
var embedded embed.FS

// Dist 返回嵌入文件系统的子文件系统，指向 dist 目录
// 这允许HTTP服务器直接服务嵌入的前端资源
//
// 返回:
//   - fs.FS: 指向dist目录的文件系统
//   - error: 如果获取子文件系统失败，返回错误
func Dist() (fs.FS, error) {
	// 使用 fs.Sub 获取 embedded 中 dist 子目录的文件系统
	return fs.Sub(embedded, "dist")
}
