package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/go-sql-driver/mysql"

	"github.com/square/blip"
)

type Secret struct {
	name   string
	client *secretsmanager.SecretsManager
}

func NewSecret(name string, sess *session.Session) Secret {
	return Secret{
		name:   name,
		client: secretsmanager.New(sess),
	}
}

func (s Secret) Get(ctx context.Context) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(s.name),
		VersionStage: aws.String("AWSCURRENT"),
	}

	sv, err := s.client.GetSecretValueWithContext(ctx, input)
	if err != nil {
		return "", fmt.Errorf("Secrets Manager API error: %s", err)
	}
	blip.Debug("DEBUG: aws secret: %+v", *sv)

	if sv.SecretString == nil || *sv.SecretString == "" {
		return "", fmt.Errorf("secret string is nil or empty")
	}

	// We store secret in secret string as JSON with "username" and "password" fields
	var v map[string]interface{}
	if err := json.Unmarshal([]byte(*sv.SecretString), &v); err != nil {
		return "", fmt.Errorf("cannot decode secret string as map[string]string: %s", err)
	}
	if v == nil {
		return "", fmt.Errorf("secret value is 'null' literal")
	}

	return v["password"].(string), nil
}

func (s Secret) SwapDSN(ctx context.Context, currentDSN string) string {
	// Only return new DSN on success and password is different. Else, return
	// an empty string which makes the hotswap driver return the original driver
	// error, i.e. it's like this func was never called. Only when this func
	// returns a non-empty string does the hotswap driver use it to swap out
	// the low-level MySQL connection.

	t0 := time.Now()
	blip.Debug("Rotating password %s...", s.name)
	ok := false
	defer func() {
		if ok {
			blip.Debug("Rotating password %s successful, took %d ms", s.name, time.Now().Sub(t0).Milliseconds())
		} else {
			// @todo glog.Warningf("Rotating password %s failed (see previous logs), took %d ms", s.name, time.Now().Sub(t0).Milliseconds())
		}
	}()

	var newPassword string
	var err error

	blip.Debug("DEBUG: currentDSN = %s", currentDSN)

	// Secret name "local" is a little hack to test locally (on our laptops)
	// because we can't test the real AWS Secrets Manager locally.
	if s.name == "local" {
		newPassword = os.Getenv("NEW_PASSWORD")
		blip.Debug("DEBUG: newPassword = NEW_PASSWORD = %s", newPassword)
	} else {
		newPassword, err = s.Get(ctx)
		if err != nil {
			// @todo glog.Errorf("Error rotating password %s: error getting secret: %s", s.name, err)
			return ""
		}
	}

	cfg, err := mysql.ParseDSN(currentDSN)
	if err != nil {
		// @todo glog.Errorf("Error rotating password %s: mysql.ParseDSN() error: %s", s.name, err)
		return ""
	}
	blip.Debug("DEBUG: old dsn: %+v", cfg)

	if cfg.Passwd == newPassword {
		// @todo glog.Warningf("Password %s has not changed", s.name)
		return ""
	}

	cfg.Passwd = newPassword
	ok = true
	blip.Debug("DEBUG: new dsn: %+v", cfg)

	return cfg.FormatDSN()
}
