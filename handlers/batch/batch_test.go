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

func getUpdates(w *httptest.ResponseRecorder) []progressUpdate {
	body := w.Body.String()
	lines := strings.SplitAfter(body, "}")
	lines = lines[:len(lines) - 1] // Remove trailing "" entry

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

	updates := getUpdates(w)
	
	assert.Equal(t, "Starting", updates[0].Status)
	assert.Equal(t, "Complete", updates[1].Status)
}

func Test_Do_SingleInstance(t *testing.T) {
	insts, servers := setupServers(handlerFactory(200))
	defer shutdownServers(servers)

	w := httptest.NewRecorder()
	proc := NewProcessor(w, insts)
	err := proc.Do("GET", nil, "", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, insts, proc.instances)

	updates := getUpdates(w)

	keyUpdate := progressUpdate{"OK", "", 200, insts[0] + "/", 0, 1}
	
	assert.Equal(t, keyUpdate, updates[1])
}

func Test_Do_Multiple(t *testing.T) {
	h := handlerFactory(200)
	insts, servers := setupServers(h, h, h)
	defer shutdownServers(servers)

	w := httptest.NewRecorder()
	proc := NewProcessor(w, insts)
	err := proc.Do("GET", nil, "", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, insts, proc.instances)

	updates := getUpdates(w)

	keyUpdate := progressUpdate{"OK", "", 200, insts[0] + "/", 0, 1}
	
	assert.Equal(t, keyUpdate, updates[1])
}