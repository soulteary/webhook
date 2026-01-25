module github.com/soulteary/webhook

go 1.25

require (
	github.com/BurntSushi/toml v1.6.0
	github.com/clbanning/mxj/v2 v2.7.0
	github.com/dustin/go-humanize v1.0.1
	github.com/fsnotify/fsnotify v1.9.0
	github.com/go-chi/chi/v5 v5.2.4
	github.com/google/uuid v1.6.0
	github.com/invopop/yaml v0.3.1
	github.com/nicksnyder/go-i18n/v2 v2.6.1
	github.com/prometheus/client_golang v1.23.2
	github.com/soulteary/cli-kit v1.1.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/sys v0.40.0
	golang.org/x/text v0.33.0
	golang.org/x/time v0.14.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/klauspost/compress v1.18.3 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/soulteary/cli-kit => ../kits/cli-kit
