package jurisdiction

import (
	"fmt"
	"sync"
)

type Profile struct {
	CountryCode string   `json:"country_code"`
	Name        string   `json:"name"`
	Formats     []string `json:"formats"`
}

type Registry struct {
	mu       sync.RWMutex
	profiles map[string]Profile
}

func NewRegistry() *Registry {
	r := &Registry{profiles: map[string]Profile{}}
	r.Register(Profile{CountryCode: "FR", Name: "Factur-X EN16931", Formats: []string{"FACTUR-X", "CII", "UBL"}})
	r.Register(Profile{CountryCode: "EU", Name: "PEPPOL BIS Billing", Formats: []string{"UBL"}})
	r.Register(Profile{CountryCode: "EU-ViDA", Name: "ViDA / EN16931 (2028+)", Formats: []string{"CII", "UBL"}})
	return r
}

func (r *Registry) Register(profile Profile) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles[profile.CountryCode] = profile
}

func (r *Registry) Get(countryCode string) (Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	profile, ok := r.profiles[countryCode]
	if !ok {
		return Profile{}, fmt.Errorf("unsupported jurisdiction %q", countryCode)
	}
	return profile, nil
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.profiles))
	for name := range r.profiles {
		out = append(out, name)
	}
	return out
}
