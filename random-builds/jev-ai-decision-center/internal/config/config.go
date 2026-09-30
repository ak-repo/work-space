package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr string
	DataFile string
	Jev      JevConfig
	Policy   PolicyConfig
}

type JevConfig struct {
	Mode    string
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

type PolicyConfig struct {
	MinCategoryConfidence float64
	MinChoiceMargin       float64
	EscalationProbability float64
}

func Load() Config {
	return Config{
		HTTPAddr: env("HTTP_ADDR", ":8081"),
		DataFile: env("DATA_FILE", "data/state.json"),
		Jev: JevConfig{
			Mode:    env("JEV_MODE", "mock"),
			BaseURL: env("JEV_BASE_URL", "https://api.typesafe.ai"),
			APIKey:  os.Getenv("TYPESAFE_API_KEY"),
			Model:   env("JEV_MODEL", "jev-latest"),
			Timeout: envDuration("JEV_TIMEOUT", 3*time.Second),
		},
		Policy: PolicyConfig{
			MinCategoryConfidence: envFloat("POLICY_MIN_CATEGORY_CONFIDENCE", 0.75),
			MinChoiceMargin:       envFloat("POLICY_MIN_CHOICE_MARGIN", 0.15),
			EscalationProbability: envFloat("POLICY_ESCALATION_PROBABILITY", 0.90),
		},
	}
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func envDuration(k string, d time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if x, e := time.ParseDuration(v); e == nil {
			return x
		}
	}
	return d
}
func envFloat(k string, d float64) float64 {
	if v := os.Getenv(k); v != "" {
		if x, e := strconv.ParseFloat(v, 64); e == nil {
			return x
		}
	}
	return d
}
