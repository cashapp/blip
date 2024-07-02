module blip_plugin

go 1.22.2

replace github.com/cashapp/blip => ../../

require github.com/cashapp/blip v0.0.0-00010101000000-000000000000

require (
	github.com/aws/aws-sdk-go-v2 v1.20.3 // indirect
	github.com/aws/smithy-go v1.14.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
