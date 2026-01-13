package cli

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewConfigClient_RequiresConfigPath(t *testing.T) {
	client, err := NewConfigClient(context.Background(), "", zap.NewNop(), time.Second)
	assert.Nil(t, client)
	assert.Error(t, err)
}

func TestNewConfigClient_InvalidConfig(t *testing.T) {
	file, err := os.CreateTemp("", "hub-config-*.json")
	assert.NoError(t, err)
	defer os.Remove(file.Name())

	_, err = file.WriteString("{\"mcpServers\": {}}")
	assert.NoError(t, err)
	assert.NoError(t, file.Close())

	client, err := NewConfigClient(context.Background(), file.Name(), zap.NewNop(), time.Second)
	assert.Nil(t, client)
	assert.Error(t, err)
}
