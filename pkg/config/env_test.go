package config

import (
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Backup original values
	origFatalf := logFatalf
	origUnmarshal := viperUnmarshal
	
	defer func() {
		logFatalf = origFatalf
		viperUnmarshal = origUnmarshal
	}()

	t.Run("Success loading config", func(t *testing.T) {
		viperUnmarshal = viper.Unmarshal
		logFatalf = origFatalf
		
		LoadConfig()
		
		// Verify some default values are set
		assert.NotEmpty(t, AppConfig.Environment)
		assert.Equal(t, "3000", AppConfig.Port)
		assert.Equal(t, "0.0.0.0", AppConfig.Host)
	})

	t.Run("Failure on viper.Unmarshal", func(t *testing.T) {
		viperUnmarshal = func(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
			return errors.New("mock unmarshal error")
		}
		
		logFatalf = func(format string, v ...interface{}) {
			panic("fatal: unmarshal error")
		}
		
		assert.PanicsWithValue(t, "fatal: unmarshal error", func() {
			LoadConfig()
		})
	})
}
