package version

import (
	versionkit "github.com/soulteary/version-kit"
)

// auto set by build system - can be overridden via ldflags
var (
	// Version 版本号
	Version = "dev"
	// Commit Git 提交 hash
	Commit = "unknown"
	// BuildDate 构建日期
	BuildDate = "unknown"
	// Branch Git 分支名
	Branch = ""
)

// GetVersionInfo 返回版本信息
func GetVersionInfo() *versionkit.Info {
	return versionkit.NewWithBranch(Version, Commit, BuildDate, Branch)
}

// String 返回简短的版本字符串
func String() string {
	return GetVersionInfo().String()
}

// Full 返回完整的版本信息
func Full() string {
	return GetVersionInfo().Full()
}
