package configuration

import (
	"os"

	"github.com/havoc-io/mutagen/pkg/encoding"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// Configuration represents the global Mutagen configuration.
type Configuration struct {
	// Synchronization contains parameters related to synchronization behavior.
	Synchronization struct {
		// Mode specifies the default synchronization mode.
		Mode sync.SynchronizationMode `toml:"mode"`
	} `toml:"synchronization"`

	// Ignore contains parameters related to synchronization ignore
	// specifications.
	Ignore struct {
		// Default specifies the default list of ignore specifications.
		Default []string `toml:"default"`

		// VCS specifies the VCS ignore mode.
		VCS sync.IgnoreVCSMode `toml:"vcs"`
	} `toml:"ignore"`

	// Symlink contains parameters related to symlink handling.
	Symlink struct {
		// Mode specifies the symlink mode.
		Mode sync.SymlinkMode `toml:"mode"`
	} `toml:"symlink"`

	// Watch contains parameters related to filesystem monitoring.
	Watch struct {
		// Mode specifies the file watching mode.
		Mode filesystem.WatchMode `toml:"mode"`

		// PollingInterval specifies the interval (in seconds) for poll-based
		// file monitoring. A value of 0 specifies that Mutagen's internal
		// default interval should be used.
		PollingInterval uint32 `toml:"pollingInterval"`
	} `toml:"watch"`

	// Permission contains parameters related to permission handling.
	Permission struct {
		// ExposureLevel specifies the permission exposure level to use in
		// "portable" permission propagation, with endpoint-specific
		// specifications (at any level of configuration) taking priority.
		ExposureLevel sync.PermissionExposureLevel `toml:"exposureLevel"`

		// NOTE: Alpha-specific and beta-specific permission exposure levels are
		// not currently supported at a global configuration level. There is no
		// technical reason that prevents this, but it seems like a footgun that
		// might inadvertently expose content. It's much better to have these
		// parameters be session-specific overrides that are specified alongside
		// the endpoint URLs which they'll affect.
	} `toml:"permission"`
}

// loadFromPath is the internal loading function. We keep it separate from Load
// so that we can get full test coverage using temporary files.
func loadFromPath(path string) (*Configuration, error) {
	// Create a configuration that we can decode into. We set any default values
	// here because nothing will be modified in this structure if the
	// configuration file doesn't exist.
	result := &Configuration{}

	// Attempt to load the configuration from disk. If loading fails due to the
	// path not existing, we return the blank configuration. We don't need to
	// allocate a fresh one in that case since the loader won't have touched it
	// if the file didn't exist.
	// TODO: Should we implement a caching mechanism where we run a stat call
	// and watch for filesystem modification?
	if err := encoding.LoadAndUnmarshalTOML(path, result); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	// Return the configuration.
	return result, nil
}

// Load loads the Mutagen configuration file from disk and populates a
// Configuration structure. If the Mutagen configuration file does not exist,
// this method will return a structure with the default configuration values.
// The returned structure is not re-used, so its members can be freely mutated.
func Load() (*Configuration, error) {
	return loadFromPath(filesystem.MutagenConfigurationPath)
}
