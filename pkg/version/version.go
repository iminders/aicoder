package version

// These are set at build time via -ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func String() string {
	return "aicoder " + Version + " (commit=" + Commit + " built=" + BuildDate + ")"
}
