package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/egose/s3proxy/internal/config"
	"github.com/spf13/cobra"
)

type topologyReport struct {
	Routes []topologyRoute `json:"routes"`
}

type topologyRoute struct {
	Ordinal        int            `json:"ordinal"`
	Label          string         `json:"label"`
	Parser         topologyParser `json:"parser"`
	Operations     []string       `json:"operations"`
	Destinations   []string       `json:"destinations"`
	Dispatch       string         `json:"dispatch"`
	OnMatch        string         `json:"on_match"`
	ReadPreference string         `json:"read_preference"`
}

type topologyParser struct {
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

func newRoutesCommand() *cobra.Command {
	var cfgPath string
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Print validated route topology as JSON without running the server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := loadTopologyReport(cfgPath)
			if err != nil {
				return err
			}
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return err
			}
			data = append(data, '\n')
			n, err := cmd.OutOrStdout().Write(data)
			if err == nil && n != len(data) {
				err = io.ErrShortWrite
			}
			if err != nil {
				return fmt.Errorf("write route topology: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&cfgPath, "config", "c", "", "path to config file (required)")
	cmd.MarkFlagRequired("config")
	return cmd
}

func loadTopologyReport(path string) (topologyReport, error) {
	rt, err := config.LoadFile(path)
	if err != nil {
		return topologyReport{}, err
	}
	report := topologyReport{Routes: make([]topologyRoute, 0, len(rt.Routes))}
	for i, route := range rt.Routes {
		parser := rt.Parsers[route.ParserRef]
		destinations := make([]string, 0, len(route.DestinationRefs))
		for _, ref := range route.DestinationRefs {
			destinations = append(destinations, rt.Targets[ref].Name)
		}
		report.Routes = append(report.Routes, topologyRoute{
			Ordinal:        i + 1,
			Label:          route.Name,
			Parser:         topologyParser{Label: parser.Name, Kind: string(parser.Kind)},
			Operations:     append([]string(nil), route.Operations...),
			Destinations:   destinations,
			Dispatch:       string(route.Dispatch),
			OnMatch:        string(route.OnMatch),
			ReadPreference: string(route.ReadPreference),
		})
	}
	return report, nil
}
