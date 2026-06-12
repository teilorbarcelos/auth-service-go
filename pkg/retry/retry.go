package retry

import (
	"fmt"
	"time"
)

type Config struct {
	MaxAttempts int
	Delay       time.Duration
	Factor      float64
}

var DefaultConfig = Config{
	MaxAttempts: 5,
	Delay:       1 * time.Second,
	Factor:      2.0,
}

func Do(fn func() error, cfg Config, description string) error {
	var err error
	delay := cfg.Delay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		if attempt == cfg.MaxAttempts {
			break
		}

		time.Sleep(delay)
		delay = time.Duration(float64(delay) * cfg.Factor)
	}

	return fmt.Errorf("%s falhou após %d tentativas: %w", description, cfg.MaxAttempts, err)
}
