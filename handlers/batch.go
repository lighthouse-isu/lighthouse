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

package handlers

import (
	"fmt"
	"sync"
	"runtime"
	"net/http"
	"io/ioutil"
)

type batchProcess struct {
	writer http.ResponseWriter
	instances []string
}

type batchStep struct {
	method string
	body interface{}
	endpoint string
	interpret ResponseInterpreter
}

type StepResult struct {
    Status string
    Code int
    Message string  

    progress int
    total int
}

type ResponseInterpreter func(resp *http.Response, err error) (StepResult, bool)

func NewBatchProcess(w http.ResponseWriter, instances []string) *batchProcess {
	return &batchProcess{w, instances, []batchStep{}}
}

func (this *batchProcess) AddStep(method string, body interface{}, endpoint string) *batchProcess {
	return this.AddInterprettedStep(method, body, endpoint, interpretResponseDefault)
}

func (this *batchProcess) AddInterprettedStep(method string, body interface{}, endpoint string, inter ResponseInterpreter) *batchProcess {
	step := batchStep{method, body, endpoint, inter}
	this.steps = append(this.steps, step)
	return this
}

func (this *batchProcess) Run() []string {
	var completed []string
	total := len(this.instances)

	wg := sync.WaitGroup{}
	queue := make(chan string, 1)

	wg.Add(total)
	for i, inst := range this.instances {
		go func(inst string, number int) {
			defer wg.Done()

			var result StepResult

			for _, step := range this.steps {
				dest := fmt.Sprintf("%s/%s", inst, step.endpoint)
				resp, err = runBatchRequest(step.method, dest, step.body)

				result, ok := step.interpret(resp, err)

				if !ok {
					break
				}
			}

			result.Progress = number
			result.Total = total

			body, _ := json.Marshal(result)
			this.writer.Write(body)

			// Yield to other goroutines
			runtime.Gosched()

			if ok {
				queue <- inst
			}
		}(inst, i)
	}

	go func() {
		defer wg.Done()
		for inst := range queue {
			completed = append(completed, inst)
		}
	}()

	wg.Wait()

	return completed
}

func runBatchRequest(method, dest string, body interface{}) (*http.Response, error) {
	req, err := http.NewRequest(method, dest, body)

	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(req)
}

func interpretResponseDefault(resp *http.Response, err error) (StepResult, bool) {
	if err != nil {
		return StepResult{"Error", 500, err.Error()}, false
	}

	body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return StepResult{"Error", 500, err.Error()}, false
    }

    switch {
    case resp.Status >= 200 && resp.Status <= 299:
    	return StepResult{"OK", resp.Code, string(body)}, true

	case resp.Status >= 300 && resp.Status <= 399:
		return StepResult{"Warning", resp.Code, string(body)}, true

	default:
		return StepResult{"Error", resp.Code, string(body)}, false
    }
}