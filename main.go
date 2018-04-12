package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"

	// Required for service registration
	_ "github.com/LuminalHQ/cloudcover/awsscan/scan/service"
)

var (
	services = flag.String("services", "",
		"Comma-separated list of services (default all)")
	regions = flag.String("regions", "",
		"Comma-separated list of regions (default all)")
	workers = flag.Int("speed", 64, "IPoAC carrier count")
)

func main() {
	flag.Parse()
	out := scan.Scan(getServices(), getConfigs(), *workers)
	out.OmitEmpty()
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Printf("%s\n", b)
}

func getServices() []string {
	if *services == "" {
		return nil
	}
	if !strings.Contains(*services, "no-") {
		return strings.Split(*services, ",")
	}
	include := make(map[string]bool)
	exclude := make(map[string]bool)
	for _, name := range strings.Split(*services, ",") {
		if ex := strings.TrimPrefix(name, "no-"); ex == name {
			include[name] = true
			delete(exclude, name)
		} else {
			exclude[ex] = true
			delete(include, ex)
		}
	}
	all := scan.Services()
	keep := all[:0]
	for _, name := range all {
		if !exclude[name] && (len(include) == 0 || include[name]) {
			keep = append(keep, name)
		}
	}
	return keep
}

func getConfigs() []*aws.Config {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(err)
	}
	all := scan.RelatedRegions(cfg.Region)
	var filter map[string]bool
	if *regions != "" {
		filter = make(map[string]bool, len(all))
		for _, r := range strings.Split(*regions, ",") {
			filter[r] = true
		}
	}
	cfgs := make([]*aws.Config, 0, len(all))
	for _, region := range all {
		if filter != nil {
			if !filter[region] {
				continue
			}
			delete(filter, region)
		}
		if region == cfg.Region {
			cfgs = append(cfgs, &cfg)
		} else {
			cpy := cfg.Copy()
			cpy.Region = region
			cfgs = append(cfgs, &cpy)
		}
	}
	if len(filter) > 0 {
		for region := range filter {
			panic("invalid region: " + region)
		}
	}
	return cfgs
}
