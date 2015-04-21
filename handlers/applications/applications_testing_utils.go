// Copyright 2014 Caleb Brose, Chris Fogerty, Rob Sheehy, Zach Taylor, Nick Miller
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package applications

import (
	"github.com/lighthouse/lighthouse/databases"

	"github.com/lighthouse/lighthouse/auth"
	"github.com/lighthouse/lighthouse/beacons"
)

func SetupTestingTable() {
	applications = databases.CommonTestingTable(appSchema)   // defined in applications.go
	deployments = databases.CommonTestingTable(deploySchema) // defined in applications.go
}

func TeardownTestingTable() {
	applications = nil
	deployments = nil
}

func makeDatabaseEntryFor(app applicationData) map[string]interface{} {
	return map[string]interface{}{
		"Id":                app.Id,
		"CurrentDeployment": app.CurrentDeployment,
		"Name":              app.Name,
		"Instances":         app.Instances,
	}
}

func setup() {
	SetupTestingTable()
	auth.SetupTestingTable()
	beacons.SetupTestingTable()
}

func teardown() {
	TeardownTestingTable()
	auth.TeardownTestingTable()
	beacons.TeardownTestingTable()
}
