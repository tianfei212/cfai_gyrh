package bootstrap

import "testing"

func TestDefaultSkillProvidersIncludes302GPTImage(t *testing.T) {
	found := false
	for _, provider := range defaultSkillProviders {
		if provider == "302-gpt-image" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("defaultSkillProviders must include 302-gpt-image")
	}
}
