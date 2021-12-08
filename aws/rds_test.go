package aws_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	//"github.com/stretchr/testify/assert"

	"github.com/cashapp/blip"
	blipAWS "github.com/cashapp/blip/aws"
	"github.com/cashapp/blip/test/mock"
)

func TestRDSClient(t *testing.T) {
	client := mock.RDSClient{
		Out: rds.DescribeDBInstancesOutput{
			DBInstances: []types.DBInstance{
				{
					DBInstanceIdentifier: aws.String("rds1"),
					Endpoint: &types.Endpoint{
						Address: aws.String("rds1"),
						Port:    3306,
					},
				},
				{
					DBInstanceIdentifier: aws.String("rds2"),
					Endpoint: &types.Endpoint{
						Address: aws.String("rds2"),
						Port:    3307,
					},
				},
			},
		},
	}
	f := mock.RDSClientFactory{
		MakeFunc: func(ba blip.AWS) (blipAWS.RDSClient, error) {
			return client, nil
		},
	}

	rdsLoader := blipAWS.RDSLoader{ClientFactory: f}

	cfg := blip.Config{}
	got, err := rdsLoader.Load(context.Background(), cfg)
	if err != nil {
		t.Error(err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d ConfigMonitor, expected 2", len(got))
	}
	if got[0].Hostname != "rds1:3306" {
		t.Errorf("ConfigMonitor[0].Hostname = %s, expected rds1:3306", got[0].Hostname)
	}
	if got[1].Hostname != "rds2:3307" {
		t.Errorf("ConfigMonitor[1].Hostname = %s, expected rds2:3307", got[1].Hostname)
	}
}
