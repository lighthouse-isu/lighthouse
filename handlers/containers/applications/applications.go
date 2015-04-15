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
)

var applications databases.TableInterface

var schema = databases.Schema{
	"Id":   "serial primary key",
	"Name": "text",
}

type applicationData struct {
	Id   int64
	Name string
}

func Init(reload bool) {
	if applications == nil {
		applications = databases.NewLockingTable(nil, "applications", schema)
	}

	if reload {
		applications.Reload()
	}
}

func CreateApplication(Name string) (int64, error) {
	values := make(map[string]interface{}, len(schema)-1)

	//    values["Id"] = "DEFAULT"
	values["Name"] = Name

	cols := []string{"Id"}
	opts := databases.SelectOptions{Top: 1, OrderBy: []string{"Id"}, Desc: true}

	var app applicationData

	err := applications.InsertReturn(values, cols, &opts, &app)
	if err != nil {
		return -1, err
	}

	return app.Id, err
}

func GetApplicationName(Id int64) (string, error) {
	var application applicationData
	where := databases.Filter{"Id": Id}
	var columns []string

	for k, _ := range schema {
		columns = append(columns, k)
	}

	err := applications.SelectRow(columns, where, nil, &application)

	if err != nil {
		return "", err
	}

	return application.Name, err
}

func GetApplicationId(Name string) (int64, error) {
	var application applicationData
	where := databases.Filter{"Name": Name}
	var columns []string

	for k, _ := range schema {
		columns = append(columns, k)
	}

	err := applications.SelectRow(columns, where, nil, &application)

	if err != nil {
		return -1, err
	}

	return application.Id, err
}
