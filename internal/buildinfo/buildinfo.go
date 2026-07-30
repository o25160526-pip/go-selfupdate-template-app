package buildinfo

// These values are intended to be injected by release pipelines using -ldflags.
// CurrentPublicKey and NextPublicKey implement signing-key rotation from day one.
var (
	CurrentPublicKey     = ""
	NextPublicKey        = ""
	DefaultManifestURL   = ""
	DefaultGitHubOwner   = ""
	DefaultGitHubRepo    = ""
	DefaultAzureIndexURL = ""
)

func PublicKeys() []string {
	out := make([]string, 0, 2)
	if CurrentPublicKey != "" {
		out = append(out, CurrentPublicKey)
	}
	if NextPublicKey != "" {
		out = append(out, NextPublicKey)
	}
	return out
}
