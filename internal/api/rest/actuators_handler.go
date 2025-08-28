package rest

import (
	"context"
	"time"

	"github.com/manuelarte/go-web-layout/internal/info"
)

type ActuatorsHandler struct{}

func (h ActuatorsHandler) ActuatorsHealth(
	_ context.Context,
	_ ActuatorsHealthRequestObject,
) (ActuatorsHealthResponseObject, error) {
	// TODO(manuelarte): Implement health check
	return ActuatorsHealth200JSONResponse{
		Components: nil,
		Status:     "",
	}, nil
}

func (h ActuatorsHandler) ActuatorsInfo(
	_ context.Context,
	_ ActuatorsInfoRequestObject,
) (ActuatorsInfoResponseObject, error) {
	return ActuatorsInfo200JSONResponse{
		App: InfoApp{
			Description: "Example of web project layout",
			Name:        "Go-Web-Layout",
			Version:     info.Version,
		},
		Git: InfoGit{
			Branch:    info.Branch,
			BuildTime: formatBuildTime(info.BuildTime),
			BuildUrl:  info.BuildURL,
			CommitId:  info.CommitID,
		},
	}, nil
}

func formatBuildTime(ts string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05-07:00", ts)

	return t
}
