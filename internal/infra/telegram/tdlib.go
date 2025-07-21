package fetcher

import (
	"fmt"

	"github.com/ScrpTrx-Go/GoTGParse/internal/config"
	"github.com/zelenin/go-tdlib/client"
)

func NewClient(cfg config.TDLibConfig) (*client.Client, error) {

	tdlibParameters := &client.SetTdlibParametersRequest{
		UseTestDc:           cfg.UseTestDc,
		DatabaseDirectory:   cfg.DatabaseDirectory,
		FilesDirectory:      cfg.FilesDirectory,
		UseFileDatabase:     cfg.UseFileDatabase,
		UseChatInfoDatabase: cfg.UseChatInfoDatabase,
		UseMessageDatabase:  cfg.UseMessageDatabase,
		UseSecretChats:      cfg.UseSecretChats,
		ApiId:               cfg.APIID,
		ApiHash:             cfg.APIHash,
		SystemLanguageCode:  cfg.SystemLanguageCode,
		DeviceModel:         cfg.DeviceModel,
		SystemVersion:       cfg.SystemVersion,
		ApplicationVersion:  cfg.ApplicationVersion,
	}

	authorizer := client.ClientAuthorizer(tdlibParameters)
	go client.CliInteractor(authorizer)

	_, err := client.SetLogVerbosityLevel(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: int32(cfg.LogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("SetLogVerbosityLevel error: %s", err)
	}

	tdlibClient, err := client.NewClient(authorizer)
	if err != nil {
		return nil, fmt.Errorf("NewClient error: %s", err)
	}

	return tdlibClient, nil
}
