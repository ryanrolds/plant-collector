module github.com/ryanrolds/plant-collector/bridge

go 1.19

require (
	github.com/sirupsen/logrus v1.9.0
	tinygo.org/x/bluetooth v0.3.0
)

require (
	github.com/JuulLabs-OSS/cbgo v0.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/godbus/dbus/v5 v5.0.3 // indirect
	github.com/muka/go-bluetooth v0.0.0-20220830075246-0746e3a1ea53 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	golang.org/x/sys v0.0.0-20220829200755-d48e67d00261 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace tinygo.org/x/bluetooth v0.3.0 => github.com/rbaron/bluetooth v0.3.1-0.20210501180115-a5ddbbc48845
