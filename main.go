// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/version"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/util/strutil"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	a          = kingpin.New("prometheus-lightsail-sd", "Tool to generate file_sd target files for AWS Lightsail.")
	outputFile = a.Flag("output.file", "Output file for file_sd compatible file.").Default("lightsail_sd.json").String()
	refresh    = a.Flag("target.refresh", "The refresh interval (in seconds).").Default("60").Int()
	profile    = a.Flag("profile", "AWS Profile").Default("default").String()

	logger log.Logger
	sess   client.ConfigProvider

	availabilityZoneLabel = model.MetaLabelPrefix + "lightsail_availability_zone"
	blueprintIdLabel      = model.MetaLabelPrefix + "lightsail_blueprint_id"
	bundleIdLabel         = model.MetaLabelPrefix + "lightsail_bundle_id"
	instanceIdLabel       = model.MetaLabelPrefix + "lightsail_instance_id"
	nameLabel             = model.MetaLabelPrefix + "lightsail_name"
	privateIpLabel        = model.MetaLabelPrefix + "lightsail_private_ip"
	publicIpLabel         = model.MetaLabelPrefix + "lightsail_public_ip"
	stateLabel            = model.MetaLabelPrefix + "lightsail_state"
	supportCodeLabel      = model.MetaLabelPrefix + "lightsail_support_code"
	tagLabel              = model.MetaLabelPrefix + "lightsail_tag_"
)

type lightsailDiscoverer struct {
	client  *lightsail.Lightsail
	refresh int
	logger  log.Logger
	lasts   map[string]struct{}
}

func (d *lightsailDiscoverer) createTarget(srv *lightsail.Instance) *targetgroup.Group {
	// create targetgroup
	tg := &targetgroup.Group{
		Source: fmt.Sprintf("lightsail/%s", *srv.Name),
		Targets: []model.LabelSet{
			model.LabelSet{
				model.AddressLabel: model.LabelValue(*srv.PrivateIpAddress),
			},
		},
		Labels: model.LabelSet{
			model.AddressLabel:                     model.LabelValue(*srv.PrivateIpAddress),
			model.LabelName(availabilityZoneLabel): model.LabelValue(*srv.Location.AvailabilityZone),
			model.LabelName(blueprintIdLabel):      model.LabelValue(*srv.BlueprintId),
			model.LabelName(bundleIdLabel):         model.LabelValue(*srv.BundleId),
			model.LabelName(instanceIdLabel):       model.LabelValue(strings.Split(*srv.SupportCode, "/")[1]),
			model.LabelName(nameLabel):             model.LabelValue(*srv.Name),
			model.LabelName(privateIpLabel):        model.LabelValue(*srv.PrivateIpAddress),
			model.LabelName(publicIpLabel):         model.LabelValue(*srv.PublicIpAddress),
			model.LabelName(stateLabel):            model.LabelValue(*srv.State.Name),
			model.LabelName(supportCodeLabel):      model.LabelValue(*srv.SupportCode),
		},
	}

	// create tag labels
	for _, t := range srv.Tags {
		if t == nil || t.Key == nil || t.Value == nil {
			continue
		}
		name := strutil.SanitizeLabelName(*t.Key)
		tg.Labels[model.LabelName(tagLabel+name)] = model.LabelValue(*t.Value)
	}

	return tg
}

func (d *lightsailDiscoverer) getTargets() ([]*targetgroup.Group, error) {
	srvs, err := d.client.GetInstances(nil)
	if err != nil {
		return nil, err
	}

	level.Debug(d.logger).Log("msg", "get servers", "count", len(srvs.Instances))

	current := make(map[string]struct{})
	tgs := make([]*targetgroup.Group, len(srvs.Instances))
	for _, s := range srvs.Instances {
		tg := d.createTarget(s)
		level.Debug(d.logger).Log("msg", "server added", "source", tg.Source)
		current[tg.Source] = struct{}{}
		tgs = append(tgs, tg)
	}

	// add empty groups for servers which have been removed since the last refresh
	for k := range d.lasts {
		if _, ok := current[k]; !ok {
			level.Debug(d.logger).Log("msg", "server deleted", "source", k)
			tgs = append(tgs, &targetgroup.Group{Source: k})
		}
	}

	d.lasts = current

	return tgs, nil
}

func (d *lightsailDiscoverer) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(d.refresh) * time.Second); ; {
		tgs, err := d.getTargets()

		if err == nil {
			ch <- tgs
		}

		// wait for ticker or exit when ctx is closed
		select {
		case <-c:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func main() {
	a.HelpFlag.Short('h')

	a.Version(version.Print("prometheus-lightsail-sd"))

	logger = log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		level.Error(logger).Log("msg", err)
		return
	}

	// use aws named profile if specified, otherwise use session.SharedConfig
	if *profile != "" {
		level.Debug(logger).Log("msg", "loading profile: "+*profile)
		sess, err = session.NewSessionWithOptions(session.Options{
			Profile:           *profile,
			SharedConfigState: session.SharedConfigEnable,
		})
		if err != nil {
			level.Error(logger).Log("msg", "error creating session", "err", err)
			return
		}
	} else {
		level.Debug(logger).Log("msg", "loading shared config: "+*profile)
		sess, err = session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		})
		if err != nil {
			level.Error(logger).Log("msg", "error creating session", "err", err)
			return
		}
	}

	// create new lightsail client (lightsail.New does not return an err currently)
	lightsailClient := lightsail.New(sess)

	ctx := context.Background()

	disc := &lightsailDiscoverer{
		client:  lightsailClient,
		refresh: *refresh,
		logger:  logger,
		lasts:   make(map[string]struct{}),
	}

	sdAdapter := NewAdapter(ctx, *outputFile, "lightsailSD", disc, logger)
	sdAdapter.Run()

	<-ctx.Done()
}
