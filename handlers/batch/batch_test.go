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

package batch

import (
    "testing"

    "fmt"
    "sort"
    "strings"
    "net/http"
    "net/http/httptest"
    "encoding/json"

    "github.com/stretchr/testify/assert"
)

func handlerFactory(code int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
	}
}

func getUpdates(w *httptest.ResponseRecorder, includeStartAndEnd bool) []progressUpdate {
	body := w.Body.String()
	lines := strings.SplitAfter(body, "}")
	lines = lines[:len(lines) - 1] // Remove trailing "" entry

	if !includeStartAndEnd {
		lines = lines[1:len(lines) - 1]
	}

	updates := make([]progressUpdate, len(lines))

	jsonStr := fmt.Sprintf("[%s]", strings.Join(lines, ","))
	json.Unmarshal([]byte(jsonStr), &updates)

	return updates
}

func Test_NewProcessor(t *testing.T) {
	writer := httptest.NewRecorder()
	instances := []string{"Inst1", "Inst2", "Inst3"}

	proc := NewProcessor(writer, instances)

	assert.Equal(t, writer, proc.writer)
	assert.Equal(t, instances, proc.instances)
}

func Test_Do_Nothing(t *testing.T) {
	w := httptest.NewRecorder()
	proc := NewProcessor(w, []string{})
	err := proc.Do("JUNK", nil, "", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, w.Code)

	updates := getUpdates(w, true)
	
	assert.Equal(t, "Starting", updates[0].Status)
	assert.Equal(t, "Complete", updates[1].Status)
}

func Test_Do_SingleInstance(t *testing.T) {
	insts, servers := SetupServers(handlerFactory(200))
	defer ShutdownServers(servers)

	w := httptest.NewRecorder()
	proc := NewProcessor(w, insts)
	err := proc.Do("GET", nil, "", nil)

	assert.Nil(t, err)
	assert.Equal(t, insts, proc.instances)

	updates := getUpdates(w, false)

	keyUpdate := progressUpdate{"OK", "", 200, insts[0] + "/", 0, 1}
	
	assert.Equal(t, keyUpdate, updates[0])
}

func Test_Do_SingleInstance_Fail(t *testing.T) {
	insts, servers := SetupServers(handlerFactory(400))
	defer ShutdownServers(servers)

	w := httptest.NewRecorder()
	proc := NewProcessor(w, insts)
	err := proc.Do("GET", nil, "", nil)

	assert.NotNil(t, err)
	assert.Equal(t, 0, len(proc.instances))

	updates := getUpdates(w, false)

	keyUpdate := progressUpdate{"Error", "", 400, insts[0] + "/", 0, 1}
	assert.Equal(t, keyUpdate, updates[0])
}

func Test_Do_Multiple(t *testing.T) {
	h := handlerFactory(200)
	insts, servers := SetupServers(h, h, h)
	defer ShutdownServers(servers)

	w := httptest.NewRecorder()
	proc := NewProcessor(w, insts)
	err := proc.Do("GET", nil, "", nil)

	assert.Nil(t, err)

	updates := getUpdates(w, false)

	for _, update := range updates {
		assert.Equal(t, "OK", update.Status)
		assert.Equal(t, insts[update.Item] + "/", update.Endpoint)
		assert.Equal(t, 200, update.Code)
	}

	sort.Strings(proc.instances)
	assert.Equal(t, insts, proc.instances)
}

func Test_Do_Multiple_Mixed(t *testing.T) {
	insts, servers := SetupServers(
		handlerFactory(200), 
		handlerFactory(500),
		handlerFactory(300),
	)
	
	defer ShutdownServers(servers)

	w := httptest.NewRecorder()
	proc := NewProcessor(w, insts)
	err := proc.Do("GET", nil, "", nil)

	assert.NotNil(t, err)
	assert.Equal(t, 2, len(proc.instances))

	updates := getUpdates(w, false)

	for _, update := range updates {
		if update.Item == 0 {
			assert.Equal(t, "OK", update.Status)
			assert.Equal(t, 200, update.Code)
		} else if update.Item == 1 {
			assert.Equal(t, "Error", update.Status)
			assert.Equal(t, 500, update.Code)
		} else {
			assert.Equal(t, "Warning", update.Status)
			assert.Equal(t, 300, update.Code)
		}

		assert.Equal(t, insts[update.Item] + "/", update.Endpoint)
	}

	sort.Strings(proc.instances)
	assert.Equal(t, insts[0], proc.instances[0])
	assert.Equal(t, insts[2], proc.instances[1])
}

func Test_Failures(t *testing.T) {
	p := handlerFactory(200)
	f := handlerFactory(500)
	insts, servers := SetupServers(p, f, f, p, f)
	defer ShutdownServers(servers)

	w := httptest.NewRecorder()
	proc := NewProcessor(w, insts)
	proc.Do("GET", nil, "", nil)

	failures := proc.Failures()

	assert.Equal(t, 3, len(failures))

	sort.Strings(failures)
	assert.Equal(t, insts[1], failures[0])
	assert.Equal(t, insts[2], failures[1])
	assert.Equal(t, insts[4], failures[2])
}