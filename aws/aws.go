// Copyright 2022 Block, Inc.

package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"

	"github.com/cashapp/blip"
)

type ConfigFactory struct {
	region string
}

func (f *ConfigFactory) Make(ba blip.AWS) (aws.Config, error) {
	if ba.Region == "auto" {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		var err error
		ba.Region, err = Region(ctx)
		if err != nil {
			blip.Debug("cannot auto-detect region: %s", err)
			return aws.Config{}, fmt.Errorf("cannot auto-detect AWS region (EC2 IMDS query failed)")
		}
		if f.region == "" {
			f.region = ba.Region
		}
	}
	if ba.Region == "" && f.region != "" {
		ba.Region = f.region
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return config.LoadDefaultConfig(ctx, config.WithRegion(ba.Region))
}

// Region auto-detects the region. Currently, the function relies on IMDS v2:
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html
// If the region cannot be detect, it returns an empty string.
func Region(ctx context.Context) (string, error) {
	blip.Debug("auto-detect AWS region")
	client := imds.New(imds.Options{})
	ec2, err := client.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return "", err
	}
	return ec2.Region, nil
}
