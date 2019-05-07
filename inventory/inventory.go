//  Copyright 2017 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

// Package inventory scans the current inventory (patches and package installed and available)
// and writes them to Guest Attributes.
package inventory

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/osconfig/attributes"
	"github.com/GoogleCloudPlatform/osconfig/config"
	"github.com/GoogleCloudPlatform/osconfig/logger"
	"github.com/GoogleCloudPlatform/osconfig/tasker"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/osinfo"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
)

const (
	inventoryURL = config.ReportURL + "/guestInventory"
)

// InstanceInventory is an instances inventory data.
type InstanceInventory struct {
	Hostname             string
	LongName             string
	ShortName            string
	Version              string
	Architecture         string
	KernelVersion        string
	OSConfigAgentVersion string
	InstalledPackages    packages.Packages
	PackageUpdates       packages.Packages
}

func write(state *InstanceInventory, url string) {
	logger.Infof("Writing instance inventory.")

	if err := attributes.PostAttribute(url+"/LastUpdated", strings.NewReader(time.Now().UTC().Format(time.RFC3339))); err != nil {
		logger.Errorf("postAttribute error: %v", err)
	}

	e := reflect.ValueOf(state).Elem()
	t := e.Type()
	for i := 0; i < e.NumField(); i++ {
		f := e.Field(i)
		u := fmt.Sprintf("%s/%s", url, t.Field(i).Name)
		logger.Debugf("postAttribute %s: %+v", u, f)
		switch f.Kind() {
		case reflect.String:
			if err := attributes.PostAttribute(u, strings.NewReader(f.String())); err != nil {
				logger.Errorf("postAttribute error: %v", err)
			}
		case reflect.Struct:
			if err := attributes.PostAttributeCompressed(u, f.Interface()); err != nil {
				logger.Errorf("postAttributeCompressed error: %v", err)
			}
		}
	}
}

// Get generates inventory data.
func Get() *InstanceInventory {
	logger.Infof("Gathering instance inventory.")

	hs := &InstanceInventory{}

	hn, err := os.Hostname()
	if err != nil {
		logger.Errorf("os.Hostname() error: %v", err)
	}

	hs.Hostname = hn

	di, err := osinfo.GetDistributionInfo()
	if err != nil {
		logger.Errorf("osinfo.GetDistributionInfo() error: %v", err)
	}

	hs.LongName = di.LongName
	hs.ShortName = di.ShortName
	hs.Version = di.Version
	hs.KernelVersion = di.Kernel
	hs.Architecture = di.Architecture
	hs.OSConfigAgentVersion = config.Version()

	var errs []string
	hs.InstalledPackages, errs = packages.GetInstalledPackages()
	if len(errs) != 0 {
		logger.Errorf("packages.GetInstalledPackages() error: %v", err)
	}

	hs.PackageUpdates, errs = packages.GetPackageUpdates()
	if len(errs) != 0 {
		logger.Errorf("packages.GetPackageUpdates() error: %v", err)
	}

	return hs
}

// Run gathers and records inventory information using tasker.Enqueue.
func Run() {
	tasker.Enqueue("Run OSInventory", func() { write(Get(), inventoryURL) })
}
