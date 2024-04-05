package flags

func Parse() AppFlags {
	envs := ParseEnvs()
	cli := ParseCLI(envs)
	return cli
}
