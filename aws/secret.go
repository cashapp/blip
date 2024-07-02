// Copyright 2024 Block, Inc.

package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/cashapp/blip"
)

type Secret struct {
	name   string
	client *secretsmanager.Client
}

func NewSecret(name string, cfg aws.Config) Secret {
	return Secret{
		name:   name,
		client: secretsmanager.NewFromConfig(cfg),
	}
}

func (s Secret) Password(ctx context.Context) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(s.name),
		VersionStage: aws.String("AWSCURRENT"),
	}

	sv, err := s.client.GetSecretValue(ctx, input)
	if err != nil {
		return "", fmt.Errorf("Secrets Manager API error: %s", err)
	}
	blip.Debug("DEBUG: aws secret: %+v", *sv)

	if sv.SecretString == nil || *sv.SecretString == "" {
		return "", fmt.Errorf("secret string is nil or empty")
	}

	var v map[string]interface{}
	if err := json.Unmarshal([]byte(*sv.SecretString), &v); err != nil {
		return "", fmt.Errorf("cannot decode secret string as map[string]string: %s", err)
	}
	if v == nil {
		return "", fmt.Errorf("secret value is 'null' literal")
	}

	return v["password"].(string), nil
}
