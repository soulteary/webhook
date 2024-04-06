# 配置参数

程序支持两种调用方法，分别是“通过命令行参数”和“设置环境变量”。

关于命令行参数，只需要记得使用 `--help`，即可查看所有的支持设置参数。

而关于环境变量的设置，我们可以通过查看 [internal/flags/define.go](https://github.com/soulteary/webhook/blob/main/internal/flags/define.go) 中的配置项，来完成一致的程序行为设置。
