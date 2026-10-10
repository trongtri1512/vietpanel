// Package license provides enterprise license validation and feature gating.
// In community mode all features are enabled; enterprise mode requires a valid
// license key to unlock premium features (resource limits, branding, support).
package license

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Tier int

const (
	TierCommunity  Tier = iota // Unlimited users/domains, basic features
	TierEnterprise              // Priority support, branding removal, advanced features
)

type License struct {
	Key       string    `json:"key"`
	Tier      Tier      `json:"tier"`
	ExpiresAt time.Time `json:"expires_at"`
	Domains   int       `json:"max_domains"`
	Users     int       `json:"max_users"`
}

var activeLicense *License

func Init() {
	if key := os.Getenv("VIETPANEL_LICENSE_KEY"); key != "" {
		if lic, err := Validate(key); err == nil {
			activeLicense = lic
		}
	}
	if activeLicense == nil {
		activeLicense = &License{
			Key:       "community",
			Tier:      TierCommunity,
			ExpiresAt: time.Now().AddDate(100, 0, 0),
			Domains:   0, // unlimited
			Users:     0, // unlimited
		}
	}
}

func Active() *License { return activeLicense }

func IsEnterprise() bool { return activeLicense != nil && activeLicense.Tier == TierEnterprise }

func Validate(key string) (*License, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("empty license key")
	}
	parts := strings.SplitN(key, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid license format")
	}
	expectedSig := sign(parts[0] + ":" + parts[1])
	if parts[2] != expectedSig[:16] {
		return nil, fmt.Errorf("invalid license signature")
	}
	var tier Tier
	switch parts[0] {
	case "enterprise":
		tier = TierEnterprise
	default:
		tier = TierCommunity
	}
	exp, err := time.Parse("2006-01-02", parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid expiry date")
	}
	return &License{
		Key:       key,
		Tier:      tier,
		ExpiresAt: exp,
	}, nil
}

func sign(data string) string {
	h := sha256.Sum256([]byte("vietpanel-salt-v1:" + data))
	return hex.EncodeToString(h[:])
}

func FeatureEnabled(feature string) bool {
	if activeLicense == nil || activeLicense.Tier == TierEnterprise {
		return true
	}
	return isCommunityFeature(feature)
}

func isCommunityFeature(f string) bool {
	communityFeatures := map[string]bool{
		"websites": true, "domains": true, "databases": true,
		"files": true, "backups": true, "dns": true,
		"emails": true, "ftp": true, "crons": true,
		"services": true, "php": true, "security": true,
		"diskusage": true,
	}
	return communityFeatures[f]
}

// GenerateLicenseKey creates a signed license key for distribution.
func GenerateLicenseKey(tier string, expiry time.Time) string {
	data := tier + ":" + expiry.Format("2006-01-02")
	sig := sign(data)
	return data + ":" + sig[:16]
}

// LicenseInfo returns JSON-serializable license status for API endpoints.
func LicenseInfo() map[string]any {
	lic := Active()
	info := map[string]any{
		"tier":        "community",
		"is_enterprise": false,
		"expires_at":  lic.ExpiresAt.Format("2006-01-02"),
	}
	if lic.Tier == TierEnterprise {
		info["tier"] = "enterprise"
		info["is_enterprise"] = true
	}
	return info
}

func init() {
	// Register license info JSON for API embedding
	_ = json.Marshal // keep json import
}