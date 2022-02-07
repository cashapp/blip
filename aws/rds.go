// Copyright 2022 Block, Inc.

package aws

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/go-sql-driver/mysql"

	"github.com/cashapp/blip"
)

type RDSClient interface {
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/rds#DescribeDBInstancesAPIClient
	rds.DescribeDBInstancesAPIClient
}

type RDSClientFactory interface {
	Make(blip.AWS) (RDSClient, error)
}

type rdsClientFactory struct {
	awsMaker blip.AWSConfigFactory
}

func NewRDSClientFactory(awsMaker blip.AWSConfigFactory) rdsClientFactory {
	return rdsClientFactory{
		awsMaker: awsMaker,
	}
}

func (f rdsClientFactory) Make(ba blip.AWS) (RDSClient, error) {
	awsCfg, err := f.awsMaker.Make(ba)
	if err != nil {
		return nil, err
	}
	client := rds.NewFromConfig(awsCfg)
	return client, nil
}

// --------------------------------------------------------------------------

type RDSLoader struct {
	ClientFactory RDSClientFactory
}

// Load calls DescribeDBInstances to return a list of RDS instances.
// This is equivalent to the command "aws rds describe-db-instances".
func (rl RDSLoader) Load(ctx context.Context, cfg blip.Config) ([]blip.ConfigMonitor, error) {
	loaderCfg := cfg.MonitorLoader.AWS
	if len(loaderCfg.Regions) == 0 {
		return nil, fmt.Errorf("no regions specififed")
	}

	mons := []blip.ConfigMonitor{}

	for _, region := range loaderCfg.Regions {
		client, err := rl.ClientFactory.Make(blip.AWS{Region: region})
		if err != nil {
			return nil, err
		}

		var marker string // pagination
	PAGES:
		for {
			// Query RDS API to list all db instances. Marker (for pagination) is set
			// (not an empty string) by previous calls--see below.
			out, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{Marker: &marker})
			if err != nil {
				return nil, err
			}

			// Make a blip.ConfigMonitor for every RDS instances
			blip.Debug("%d db instances", len(out.DBInstances))
			for _, instance := range out.DBInstances {
				// During provision or decommission, endpoint can be nil
				if instance.Endpoint == nil || instance.Endpoint.Address == nil {
					blip.Debug("%s endpoint nil (status=%v)", *instance.DBInstanceIdentifier, *instance.DBInstanceStatus)
					continue
				}
				monCfg := blip.ConfigMonitor{
					MonitorId: *instance.DBInstanceIdentifier,
					Hostname:  fmt.Sprintf("%s:%d", *instance.Endpoint.Address, instance.Endpoint.Port),
					AWS: blip.ConfigAWS{
						Region: region,
					},
				}
				mons = append(mons, monCfg)
				blip.Debug("loaded %si dbid=%v cluster=%v version=%v az=%v stauts=%v ",
					monCfg.Hostname, *instance.DBInstanceIdentifier, *instance.DBClusterIdentifier, *instance.EngineVersion, *instance.AvailabilityZone, *instance.DBInstanceStatus)
			}

			// Max 100 instances per page; read next page if marker is set
			if out.Marker != nil {
				blip.Debug("next page")
				marker = *out.Marker
				continue PAGES // next page
			}

			break PAGES // last page
		}
	}

	return mons, nil
}

var once sync.Once

// RegisterRDSCA registers the Amazon RDS certificate authority (CA) to enable
// TLS connections to RDS. The TLS param is called "rds". It is only registered
// once (as required by Go), but it's safe to call multiple times.
func RegisterRDSCA() {
	once.Do(func() {
		blip.Debug("loading RDS CA")
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(rds2019rootCA)
		tlsConfig := &tls.Config{RootCAs: caCertPool}
		mysql.RegisterTLSConfig("rds", tlsConfig)
	})
}

// rds-ca-2019-root.pem
var rds2019rootCA = []byte(`-----BEGIN CERTIFICATE-----
MIIEBjCCAu6gAwIBAgIJAMc0ZzaSUK51MA0GCSqGSIb3DQEBCwUAMIGPMQswCQYD
VQQGEwJVUzEQMA4GA1UEBwwHU2VhdHRsZTETMBEGA1UECAwKV2FzaGluZ3RvbjEi
MCAGA1UECgwZQW1hem9uIFdlYiBTZXJ2aWNlcywgSW5jLjETMBEGA1UECwwKQW1h
em9uIFJEUzEgMB4GA1UEAwwXQW1hem9uIFJEUyBSb290IDIwMTkgQ0EwHhcNMTkw
ODIyMTcwODUwWhcNMjQwODIyMTcwODUwWjCBjzELMAkGA1UEBhMCVVMxEDAOBgNV
BAcMB1NlYXR0bGUxEzARBgNVBAgMCldhc2hpbmd0b24xIjAgBgNVBAoMGUFtYXpv
biBXZWIgU2VydmljZXMsIEluYy4xEzARBgNVBAsMCkFtYXpvbiBSRFMxIDAeBgNV
BAMMF0FtYXpvbiBSRFMgUm9vdCAyMDE5IENBMIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEArXnF/E6/Qh+ku3hQTSKPMhQQlCpoWvnIthzX6MK3p5a0eXKZ
oWIjYcNNG6UwJjp4fUXl6glp53Jobn+tWNX88dNH2n8DVbppSwScVE2LpuL+94vY
0EYE/XxN7svKea8YvlrqkUBKyxLxTjh+U/KrGOaHxz9v0l6ZNlDbuaZw3qIWdD/I
6aNbGeRUVtpM6P+bWIoxVl/caQylQS6CEYUk+CpVyJSkopwJlzXT07tMoDL5WgX9
O08KVgDNz9qP/IGtAcRduRcNioH3E9v981QO1zt/Gpb2f8NqAjUUCUZzOnij6mx9
McZ+9cWX88CRzR0vQODWuZscgI08NvM69Fn2SQIDAQABo2MwYTAOBgNVHQ8BAf8E
BAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUc19g2LzLA5j0Kxc0LjZa
pmD/vB8wHwYDVR0jBBgwFoAUc19g2LzLA5j0Kxc0LjZapmD/vB8wDQYJKoZIhvcN
AQELBQADggEBAHAG7WTmyjzPRIM85rVj+fWHsLIvqpw6DObIjMWokpliCeMINZFV
ynfgBKsf1ExwbvJNzYFXW6dihnguDG9VMPpi2up/ctQTN8tm9nDKOy08uNZoofMc
NUZxKCEkVKZv+IL4oHoeayt8egtv3ujJM6V14AstMQ6SwvwvA93EP/Ug2e4WAXHu
cbI1NAbUgVDqp+DRdfvZkgYKryjTWd/0+1fS8X1bBZVWzl7eirNVnHbSH2ZDpNuY
0SBd8dj5F6ld3t58ydZbrTHze7JJOd8ijySAp4/kiu9UfZWuTPABzDa/DSdz9Dk/
zPW4CXXvhLmE02TA9/HeCw3KEHIwicNuEfw=
-----END CERTIFICATE-----`)
