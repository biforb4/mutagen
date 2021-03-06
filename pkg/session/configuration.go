package session

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/configuration"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// ConfigurationSource represents the source of a configuration object.
type ConfigurationSource uint8

const (
	// ConfigurationSourceSession specifies that a configuration object came
	// from a session object stored on disk.
	ConfigurationSourceSession ConfigurationSource = iota
	// ConfigurationSourceGlobal specifies that a configuration object was
	// loaded from the global configuration file.
	ConfigurationSourceGlobal
	// ConfigurationSourceCreate specifies that a configuration object came from
	// a create RPC request.
	ConfigurationSourceCreate
)

// EnsureValid ensures that Configuration's invariants are respected.
func (c *Configuration) EnsureValid(source ConfigurationSource) error {
	// A nil configuration is not considered valid.
	if c == nil {
		return errors.New("nil configuration")
	}

	// Verify that the synchronization mode is unspecified or supported for
	// usage.
	if !c.SynchronizationMode.IsDefault() && !c.SynchronizationMode.Supported() {
		return errors.New("unknown or unsupported synchronization mode")
	}

	// Verify that the symlink mode is unspecified or supported for usage.
	if c.SymlinkMode != sync.SymlinkMode_SymlinkDefault && !c.SymlinkMode.Supported() {
		return errors.New("unknown or unsupported symlink mode")
	}

	// Verify that the watch mode is unspecified or supported for usage.
	if c.WatchMode != filesystem.WatchMode_WatchDefault && !c.WatchMode.Supported() {
		return errors.New("unknown or unsupported watch mode")
	}

	// The watch polling interval doesn't need to be validated - any of its
	// values are technically valid.

	// Verify that default ignores are allowed to be specified and that all
	// specified default ignores are valid.
	if source == ConfigurationSourceCreate && len(c.DefaultIgnores) > 0 {
		return errors.New("create configuration with default ignores specified")
	}
	for _, ignore := range c.DefaultIgnores {
		if !sync.ValidIgnorePattern(ignore) {
			return errors.Errorf("invalid default ignore pattern: %s", ignore)
		}
	}

	// Verify that ignores are allowed to be specified and that all specified
	// ignores are valid.
	if source == ConfigurationSourceGlobal && len(c.Ignores) > 0 {
		return errors.New("global configuration with ignores specified")
	}
	for _, ignore := range c.Ignores {
		if !sync.ValidIgnorePattern(ignore) {
			return errors.Errorf("invalid ignore pattern: %s", ignore)
		}
	}

	// Verify that the VCS ignore mode is unspecified or supported for usage.
	if c.IgnoreVCSMode != sync.IgnoreVCSMode_IgnoreVCSDefault && !c.IgnoreVCSMode.Supported() {
		return errors.New("unknown or unsupported VCS ignore mode")
	}

	// Verify that permission exposure levels are unspecified or supported for
	// usage.
	if !c.PermissionExposureLevel.IsDefault() && !c.PermissionExposureLevel.Supported() {
		return errors.New("unknown or unsupported permission exposure level")
	}
	if source == ConfigurationSourceGlobal {
		if !c.AlphaPermissionExposureLevel.IsDefault() {
			return errors.New("global configuration with alpha-specific permission exposure level specified")
		} else if !c.BetaPermissionExposureLevel.IsDefault() {
			return errors.New("global configuration with beta-specific permission exposure level specified")
		}
	} else {
		if !c.AlphaPermissionExposureLevel.IsDefault() && !c.AlphaPermissionExposureLevel.Supported() {
			return errors.New("unknown or unsupported alpha permission exposure level")
		} else if !c.BetaPermissionExposureLevel.IsDefault() && !c.BetaPermissionExposureLevel.Supported() {
			return errors.New("unknown or unsupported beta permission exposure level")
		}
	}

	// Success.
	return nil
}

// snapshotGlobalConfiguration loads the global configuration, transfers the
// relevant parameters to a session configuration, and returns the resulting
// value. It does not validate the
func snapshotGlobalConfiguration() (*Configuration, error) {
	// Load the global configuration.
	configuration, err := configuration.Load()
	if err != nil {
		return nil, errors.Wrap(err, "unable to load global configuration")
	}

	// Create a session configuration object.
	result := &Configuration{
		SynchronizationMode:     configuration.Synchronization.Mode,
		SymlinkMode:             configuration.Symlink.Mode,
		WatchMode:               configuration.Watch.Mode,
		WatchPollingInterval:    configuration.Watch.PollingInterval,
		DefaultIgnores:          configuration.Ignore.Default,
		IgnoreVCSMode:           configuration.Ignore.VCS,
		PermissionExposureLevel: configuration.Permission.ExposureLevel,
	}

	// Verify that the resulting configuration is valid.
	if err := result.EnsureValid(ConfigurationSourceGlobal); err != nil {
		return nil, errors.Wrap(err, "global configuration invalid")
	}

	// Success.
	return result, nil
}

// MergeConfigurations merges a per-session and global configuration, allowing
// the per-session configuration to merge with or override the global
// configuration.
func MergeConfigurations(session, global *Configuration) *Configuration {
	// Create the resulting configuration.
	result := &Configuration{}

	// Merge synchronization mode.
	if !session.SynchronizationMode.IsDefault() {
		result.SynchronizationMode = session.SynchronizationMode
	} else {
		result.SynchronizationMode = global.SynchronizationMode
	}

	// Merge symlink mode.
	if session.SymlinkMode != sync.SymlinkMode_SymlinkDefault {
		result.SymlinkMode = session.SymlinkMode
	} else {
		result.SymlinkMode = global.SymlinkMode
	}

	// Merge watch mode.
	if session.WatchMode != filesystem.WatchMode_WatchDefault {
		result.WatchMode = session.WatchMode
	} else {
		result.WatchMode = global.WatchMode
	}

	// Merge polling interval.
	if session.WatchPollingInterval != 0 {
		result.WatchPollingInterval = session.WatchPollingInterval
	} else {
		result.WatchPollingInterval = global.WatchPollingInterval
	}

	// Propagate default ignores.
	result.DefaultIgnores = global.DefaultIgnores

	// Propagate per-session ignores.
	result.Ignores = session.Ignores

	// Merge VCS ignore mode.
	if session.IgnoreVCSMode != sync.IgnoreVCSMode_IgnoreVCSDefault {
		result.IgnoreVCSMode = session.IgnoreVCSMode
	} else {
		result.IgnoreVCSMode = global.IgnoreVCSMode
	}

	// Merge permission exposure levels. Endpoint-specific exposure levels are
	// currently disallowed in the global configuration for safety reasons.
	if !session.PermissionExposureLevel.IsDefault() {
		result.PermissionExposureLevel = session.PermissionExposureLevel
	} else {
		result.PermissionExposureLevel = global.PermissionExposureLevel
	}
	result.AlphaPermissionExposureLevel = session.AlphaPermissionExposureLevel
	result.BetaPermissionExposureLevel = session.BetaPermissionExposureLevel

	// Done.
	return result
}
