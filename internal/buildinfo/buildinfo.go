package buildinfo

// These variables will be set during build time using ldflags
var (
	// Version is the version of the application
	Version = "dev"

	// CommitHash is the git commit hash of the build
	CommitHash = "unknown"

	// BuildDate is the date when the application was built
	BuildDate = "unknown"
)

// Info represents build information
type Info struct {
	Version    string `json:"version"`
	CommitHash string `json:"commitHash"`
	BuildDate  string `json:"buildDate"`
}

// GetInfo returns the build information
func GetInfo() Info {
	return Info{
		Version:    Version,
		CommitHash: CommitHash,
		BuildDate:  BuildDate,
	}
}
