package params

import (
	"fmt"
	"runtime/debug"
)

const (
	VersionMajor = 5         // Major version component of the current release
	VersionMinor = 6         // Minor version component of the current release
	VersionPatch = 1         // Patch version component of the current release
	VersionMeta  = "mainnet" // Version metadata to append to the version string
)

// getVersion returns the base version string without metadata.
func getVersion() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}

// getVersionWithMeta returns the version string including metadata.
func getVersionWithMeta() string {
	if VersionMeta == "" {
		return getVersion()
	}
	return fmt.Sprintf("%s-%s", getVersion(), VersionMeta)
}

// ArchiveVersion returns the version string used for Geth archives.
// e.g. "1.8.11-dea1ce05" for stable releases, or "1.8.13-unstable-21c059b6" for unstable releases.
func ArchiveVersion(gitCommit string) (string, error) {
	if len(gitCommit) < 8 {
		return "", fmt.Errorf("gitCommit must be at least 8 characters long")
	}

	vsn := getVersionWithMeta()
	return fmt.Sprintf("%s-%s", vsn, gitCommit[:8]), nil
}

// VersionWithCommit returns the version string including git commit and date.
func VersionWithCommit(gitCommit, gitDate string) (string, error) {
	if len(gitCommit) < 8 {
		return "", fmt.Errorf("gitCommit must be at least 8 characters long")
	}

	vsn := getVersionWithMeta()
	vsn = fmt.Sprintf("%s-%s", vsn, gitCommit[:8])

	if VersionMeta != "stable" && gitDate != "" {
		vsn = fmt.Sprintf("%s-%s", vsn, gitDate)
	}

	return vsn, nil
}

// getCommitHash returns the current git commit hash.
func getCommitHash() (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", fmt.Errorf("could not read build info")
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value, nil
		}
	}
	return "", fmt.Errorf("vcs.revision not found in build info")
}

				return setting.Value
			}
		}
	}
	return ""
}()
