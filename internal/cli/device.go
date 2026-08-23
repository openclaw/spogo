package cli

import (
	"fmt"
	"strings"

	"github.com/steipete/spogo/internal/app"
)

type DeviceCmd struct {
	List DeviceListCmd `kong:"cmd,help='List devices.'"`
	Set  DeviceSetCmd  `kong:"cmd,help='Set active device.'"`
}

type DeviceListCmd struct{}

type DeviceSetCmd struct {
	Device string `arg:"" required:"" help:"Device name or id."`
}

func (cmd *DeviceListCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	devices, err := client.Devices(cmdCtx)
	if err != nil {
		return err
	}
	plain := make([]string, 0, len(devices))
	human := make([]string, 0, len(devices))
	for _, device := range devices {
		plain = append(plain, fmt.Sprintf("%s\t%s\t%t", device.ID, device.Name, device.Active))
		label := device.Name
		if device.Active {
			label = ctx.Output.Theme.Accent(label)
		}
		line := fmt.Sprintf("%s (%s)", label, device.Type)
		if marker := activeMarker(device.Active); marker != "" {
			line += " " + marker
		}
		human = append(human, line)
	}
	return ctx.Output.Emit(devices, plain, human)
}

func (cmd *DeviceSetCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	devices, err := client.Devices(cmdCtx)
	if err != nil {
		return err
	}
	id := cmd.Device
	for _, device := range devices {
		if strings.EqualFold(device.ID, cmd.Device) || strings.EqualFold(device.Name, cmd.Device) {
			id = device.ID
			break
		}
	}
	if err := client.Transfer(cmdCtx, id); err != nil {
		return err
	}
	return emitOK(ctx, map[string]any{"status": "ok", "device": id}, fmt.Sprintf("Switched to %s", id))
}

func activeMarker(active bool) string {
	if active {
		return "(active)"
	}
	return ""
}
