package app

import (
	"context"

	"github.com/steipete/spogo/internal/config"
)

func NewContext(settings Settings) (*Context, error) {
	configPath, err := resolveConfigPath(settings.ConfigPath)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	profileKey := resolveProfileKey(cfg, settings.Profile)
	profile := applySettings(cfg.Profile(profileKey), settings)
	writer, err := newOutputWriter(settings)
	if err != nil {
		return nil, err
	}
	return &Context{
		Settings:   settings,
		Config:     cfg,
		ConfigPath: configPath,
		Profile:    profile,
		ProfileKey: profileKey,
		Output:     writer,
		commandCtx: context.Background(),
	}, nil
}

func resolveConfigPath(configPath string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	return config.DefaultPath()
}

func resolveProfileKey(cfg *config.Config, requested string) string {
	if requested != "" {
		return requested
	}
	if cfg != nil && cfg.DefaultProfile != "" {
		return cfg.DefaultProfile
	}
	return config.DefaultProfile
}

func applySettings(profile config.Profile, settings Settings) config.Profile {
	if settings.Market != "" {
		profile.Market = settings.Market
	}
	if settings.Language != "" {
		profile.Language = settings.Language
	}
	if settings.Device != "" {
		profile.Device = settings.Device
	}
	if settings.Engine != "" {
		profile.Engine = settings.Engine
	}
	if settings.Auth != "" {
		profile.Auth = settings.Auth
	}
	if settings.SpotifyClientID != "" {
		profile.SpotifyClientID = settings.SpotifyClientID
	}
	if settings.SpotifyRedirectURI != "" {
		profile.SpotifyRedirectURI = settings.SpotifyRedirectURI
	}
	return profile
}
