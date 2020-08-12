// Copyright 2015 The Prometheus Authors
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

// +build !nobonding

package collector

import (
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
)

type bondingCollector struct {
	fs             sysfs.FS
	slaves, active typedDesc
	logger         log.Logger
}

func init() {
	registerCollector("bonding", defaultEnabled, NewBondingCollector)
}

// NewBondingCollector returns a newly allocated bondingCollector.
// It exposes the number of configured and active slave of linux bonding interfaces.
func NewBondingCollector(logger log.Logger) (Collector, error) {
	fs, err := sysfs.NewFS(*sysPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sysfs: %w", err)
	}
	return &bondingCollector{
		fs: fs,
		slaves: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bonding", "slaves"),
			"Number of configured slaves per bonding interface.",
			[]string{"master"}, nil,
		), prometheus.GaugeValue},
		active: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "bonding", "active"),
			"Number of active slaves per bonding interface.",
			[]string{"master"}, nil,
		), prometheus.GaugeValue},
		logger: logger,
	}, nil
}

// Update reads and exposes bonding states, implements Collector interface. Caution: This works only on linux.
func (c *bondingCollector) Update(ch chan<- prometheus.Metric) error {
	bondingStats, err := c.fs.NetClassBonding()
	if err != nil {
		return err
	}
	if bondingStats == nil {
		level.Debug(c.logger).Log("msg", "Not collecting bonding, no bonds found")
		return ErrNoData
	}
	for master, bondingInfo := range bondingStats {
		ch <- c.slaves.mustNewConstMetric(float64(len(bondingInfo.Slaves)), master)
		active := 0
		for _, slave := range bondingInfo.Slaves {
			if slave.MiiStatus == 1 {
				active++
			}
		}
		ch <- c.active.mustNewConstMetric(float64(active), master)
	}
	return nil
}
