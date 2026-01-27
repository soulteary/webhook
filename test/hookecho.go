// Hook Echo is a simply utility used for testing the Webhook package.

package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// RunHookecho 执行 hookecho 的主要逻辑
// 这个函数可以被测试，而 main 函数只是调用它
// writer 用于输出，如果为 nil 则使用 os.Stdout
func RunHookecho(args []string, environ []string, writer io.Writer) (shouldExit bool, exitCode int) {
	if writer == nil {
		writer = os.Stdout
	}

	if len(args) > 1 {
		_, _ = fmt.Fprintf(writer, "arg: %s\n", strings.Join(args[1:], " "))
	}

	var env []string
	for _, v := range environ {
		if strings.HasPrefix(v, "HOOK_") {
			env = append(env, v)
		}
	}

	if len(env) > 0 {
		_, _ = fmt.Fprintf(writer, "env: %s\n", strings.Join(env, " "))
	}

	if (len(args) > 1) && (strings.HasPrefix(args[1], "exit=")) {
		exit_code_str := args[1][5:]
		exit_code, err := strconv.Atoi(exit_code_str)
		if err != nil {
			_, _ = fmt.Fprintf(writer, "Exit code %s not an int!", exit_code_str)
			return true, -1
		}
		return true, exit_code
	}

	return false, 0
}

func main() {
	shouldExit, exitCode := RunHookecho(os.Args, os.Environ(), nil)
	if shouldExit {
		os.Exit(exitCode)
	}
}
