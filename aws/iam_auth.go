// Copyright 2022 Block, Inc.

package aws

import (
	"context"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
)

var portRe = regexp.MustCompile(`:\d+$`)

type AuthToken struct {
	username string
	hostname string
	cfg      aws.Config
}

func NewAuthToken(username, hostname string, cfg aws.Config) AuthToken {
	// RDS auth tokens require the :3306 suffix
	if !portRe.MatchString(hostname) {
		hostname += ":3306"
	}

	return AuthToken{
		username: username,
		hostname: hostname,
		cfg:      cfg,
	}
}

func (a AuthToken) Password(ctx context.Context) (string, error) {
	return auth.BuildAuthToken(
		ctx,
		a.hostname,
		a.cfg.Region,
		a.username,
		a.cfg.Credentials,
	)
}
