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

	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"

	"github.com/stretchr/testify/assert"

	"github.com/lighthouse/lighthouse/auth"
	"github.com/lighthouse/lighthouse/beacons"
)

func setup() *auth.User {
	beacons.SetupTestingTable()
	auth.SetupTestingTable()
	auth.CreateUser("email", "", "")
	user, _ := auth.GetUser("email")
	return user
}

func teardown() {
	beacons.TeardownTestingTable()
	auth.TeardownTestingTable()
}

func handlerFactory(code int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
	}
}

func getUpdates(w *httptest.ResponseRecorder, includeStartAndEnd bool) []progressUpdate {
	body := w.Body.String()
	lines := strings.SplitAfter(body, "}")
	lines = lines[:len(lines)-1] // Remove trailing "" entry

	if !includeStartAndEnd {
		lines = lines[1 : len(lines)-1]
	}

	updates := make([]progressUpdate, len(lines))

	jsonStr := fmt.Sprintf("[%s]", strings.Join(lines, ","))
	json.Unmarshal([]byte(jsonStr), &updates)

	return updates
}

func Test_NewProcessor(t *testing.T) {
	user := setup()
	defer teardown()

	writer := httptest.NewRecorder()
	instances := []string{"Inst1", "Inst2", "Inst3"}

	proc := NewProcessor(user, writer, instances)

	assert.Equal(t, writer, proc.writer)
	assert.Equal(t, instances, proc.instances)
}

func Test_Do_Nothing(t *testing.T) {
	user := setup()
	defer teardown()

	w := httptest.NewRecorder()
	proc := NewProcessor(user, w, []string{})
	err := proc.Do("TEST", "GET", nil, "ENDPOINT", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, w.Code)

	updates := getUpdates(w, true)

	keyStart := progressUpdate{"Starting", "GET", "ENDPOINT", "TEST", 0, "", 0, 0}
	keyComplete := progressUpdate{"Complete", "GET", "ENDPOINT", "TEST", 0, "", 0, 0}

	assert.Equal(t, keyStart, updates[0])
	assert.Equal(t, keyComplete, updates[1])
}

func Test_Finalize(t *testing.T) {
	w := httptest.NewRecorder()
	Finalize(w)

	var update progressUpdate
	json.Unmarshal(w.Body.Bytes(), &update)

	assert.Equal(t, "Finalized", update.Status)
}

func Test_Do_SingleInstance(t *testing.T) {
	user := setup()
	defer teardown()

	insts, servers := SetupServers(handlerFactory(200))
	defer ShutdownServers(servers)

	w := httptest.NewRecorder()
	proc := NewProcessor(user, w, insts)
	err := proc.Do("TEST", "GET", nil, "", nil)

	assert.Nil(t, err)
	assert.Equal(t, insts, proc.instances)

	updates := getUpdates(w, false)

	keyUpdate := progressUpdate{"OK", "GET", "", "", 200, insts[0], 0, 1}

	assert.Equal(t, keyUpdate, updates[0])
}

func Test_Do_SingleInstance_Fail(t *testing.T) {
	user := setup()
	defer teardown()

	insts, servers := SetupServers(handlerFactory(400))
	defer ShutdownServers(servers)

	w := httptest.NewRecorder()
	proc := NewProcessor(user, w, insts)
	err := proc.Do("TEST", "GET", nil, "", nil)

	assert.NotNil(t, err)
	assert.Equal(t, 0, len(proc.instances))

	updates := getUpdates(w, false)

	keyUpdate := progressUpdate{"Error", "GET", "", "", 400, insts[0], 0, 1}
	assert.Equal(t, keyUpdate, updates[0])
}

func Test_Do_Multiple(t *testing.T) {
	user := setup()
	defer teardown()

	h := handlerFactory(200)
	insts, servers := SetupServers(h, h, h)
	defer ShutdownServers(servers)

	w := httptest.NewRecorder()
	proc := NewProcessor(user, w, insts)
	err := proc.Do("TEST", "GET", nil, "", nil)

	assert.Nil(t, err)

	updates := getUpdates(w, false)

	for _, update := range updates {
		assert.Equal(t, "OK", update.Status)
		assert.Equal(t, insts[update.Item], update.Instance)
		assert.Equal(t, 200, update.Code)
	}

	sort.Strings(proc.instances)
	assert.Equal(t, insts, proc.instances)
}

func Test_Do_Multiple_Mixed(t *testing.T) {
	user := setup()
	defer teardown()

	insts, servers := SetupServers(
		handlerFactory(200),
		handlerFactory(500),
		handlerFactory(300),
	)

	defer ShutdownServers(servers)

	w := httptest.NewRecorder()
	proc := NewProcessor(user, w, insts)
	err := proc.Do("TEST", "GET", nil, "", nil)

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

		assert.Equal(t, insts[update.Item], update.Instance)
	}

	sort.Strings(proc.instances)
	assert.Equal(t, insts[0], proc.instances[0])
	assert.Equal(t, insts[2], proc.instances[1])
}

func Test_Failures(t *testing.T) {
	user := setup()
	defer teardown()

	p := handlerFactory(200)
	f := handlerFactory(500)
	insts, servers := SetupServers(p, f, f, p, f)
	defer ShutdownServers(servers)

	w := httptest.NewRecorder()
	proc := NewProcessor(user, w, insts)
	proc.Do("TEST", "GET", nil, "", nil)

	failures := proc.FailureProcessor()

	assert.Equal(t, 3, len(failures.instances))

	sort.Strings(failures.instances)
	assert.Equal(t, insts[1], failures.instances[0])
	assert.Equal(t, insts[2], failures.instances[1])
	assert.Equal(t, insts[4], failures.instances[2])
}

func Test_DefaultInterpret_Long(t *testing.T) {
	buf := make([]byte, 83)
	for i := 0; i < 83; i += 1 {
		buf[i] = byte('t')
	}

	key := make([]byte, 83)
	for i := 0; i < 80; i += 1 {
		key[i] = byte('t')
	}
	key[80] = byte('.')
	key[81] = byte('.')
	key[82] = byte('.')

	res, err := interpretResponseDefault(200, bytes.NewBuffer(buf), nil)

	assert.Nil(t, err)
	assert.Equal(t, string(key), res.Message)
}
