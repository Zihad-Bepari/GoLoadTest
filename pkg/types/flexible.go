package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// parseFlexibleDuration parses either string like "10s", "500ms" or numeric nanoseconds.
func parseFlexibleDuration(v interface{}) (time.Duration, error) {
	if v == nil {
		return 0, nil
	}
	switch val := v.(type) {
	case string:
		if val == "" {
			return 0, nil
		}
		return time.ParseDuration(val)
	case float64:
		return time.Duration(val), nil
	default:
		return 0, fmt.Errorf("invalid duration value: %v", v)
	}
}

// UnmarshalJSON provides flexible duration parsing for Config.
func (c *Config) UnmarshalJSON(data []byte) error {
	type Alias Config
	aux := struct {
		Duration interface{} `json:"duration"`
		RampUp   interface{} `json:"rampUp"`
		RampDown interface{} `json:"rampDown"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Duration != nil {
		dur, err := parseFlexibleDuration(aux.Duration)
		if err != nil {
			return fmt.Errorf("invalid config duration: %w", err)
		}
		c.Duration = dur
	}

	if aux.RampUp != nil {
		rampUp, err := parseFlexibleDuration(aux.RampUp)
		if err != nil {
			return fmt.Errorf("invalid config rampUp: %w", err)
		}
		c.RampUp = rampUp
	}

	if aux.RampDown != nil {
		rampDown, err := parseFlexibleDuration(aux.RampDown)
		if err != nil {
			return fmt.Errorf("invalid config rampDown: %w", err)
		}
		c.RampDown = rampDown
	}

	return nil
}

// UnmarshalJSON provides flexible duration parsing for EndpointSpec.
func (e *EndpointSpec) UnmarshalJSON(data []byte) error {
	type Alias EndpointSpec
	aux := struct {
		Timeout interface{} `json:"timeout"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Timeout != nil {
		timeout, err := parseFlexibleDuration(aux.Timeout)
		if err != nil {
			return fmt.Errorf("invalid endpoint timeout: %w", err)
		}
		e.Timeout = timeout
	}

	return nil
}

// UnmarshalJSON provides flexible duration parsing for Thresholds.
func (th *Thresholds) UnmarshalJSON(data []byte) error {
	type Alias Thresholds
	aux := struct {
		MaxP95        interface{} `json:"maxP95"`
		MaxP99        interface{} `json:"maxP99"`
		MaxAvgLatency interface{} `json:"maxAvgLatency"`
		*Alias
	}{
		Alias: (*Alias)(th),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.MaxP95 != nil {
		dur, err := parseFlexibleDuration(aux.MaxP95)
		if err != nil {
			return fmt.Errorf("invalid maxP95: %w", err)
		}
		th.MaxP95 = dur
	}

	if aux.MaxP99 != nil {
		dur, err := parseFlexibleDuration(aux.MaxP99)
		if err != nil {
			return fmt.Errorf("invalid maxP99: %w", err)
		}
		th.MaxP99 = dur
	}

	if aux.MaxAvgLatency != nil {
		dur, err := parseFlexibleDuration(aux.MaxAvgLatency)
		if err != nil {
			return fmt.Errorf("invalid maxAvgLatency: %w", err)
		}
		th.MaxAvgLatency = dur
	}

	return nil
}

// UnmarshalJSON provides flexible duration parsing for CircuitBreakerConfig.
func (cb *CircuitBreakerConfig) UnmarshalJSON(data []byte) error {
	type Alias CircuitBreakerConfig
	aux := struct {
		MaxLatency interface{} `json:"maxLatency"`
		*Alias
	}{
		Alias: (*Alias)(cb),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.MaxLatency != nil {
		dur, err := parseFlexibleDuration(aux.MaxLatency)
		if err != nil {
			return fmt.Errorf("invalid maxLatency: %w", err)
		}
		cb.MaxLatency = dur
	}

	return nil
}

// UnmarshalJSON provides flexible duration parsing for MonitorConfig.
func (m *MonitorConfig) UnmarshalJSON(data []byte) error {
	type Alias MonitorConfig
	aux := struct {
		Interval interface{} `json:"interval"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Interval != nil {
		dur, err := parseFlexibleDuration(aux.Interval)
		if err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}
		m.Interval = dur
	}

	return nil
}

// UnmarshalJSON provides flexible duration parsing for StressConfig.
func (s *StressConfig) UnmarshalJSON(data []byte) error {
	type Alias StressConfig
	aux := struct {
		StepInterval    interface{} `json:"stepInterval"`
		CrashMaxLatency interface{} `json:"crashMaxLatency"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.StepInterval != nil {
		dur, err := parseFlexibleDuration(aux.StepInterval)
		if err != nil {
			return fmt.Errorf("invalid stepInterval: %w", err)
		}
		s.StepInterval = dur
	}

	if aux.CrashMaxLatency != nil {
		dur, err := parseFlexibleDuration(aux.CrashMaxLatency)
		if err != nil {
			return fmt.Errorf("invalid crashMaxLatency: %w", err)
		}
		s.CrashMaxLatency = dur
	}

	return nil
}
