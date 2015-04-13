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
	"fmt"
	"sync"
	"errors"
	"runtime"
	"net/http"
	"io/ioutil"
	"encoding/json"

	"github.com/lighthouse/lighthouse/auth"
	"github.com/lighthouse/lighthouse/handlers/docker"
)

type Processor struct {
	writer http.ResponseWriter
	instances []string
	failures []string
	user *auth.User
}

type Result struct {
    Status string
    Message string
    Code int
}

type progressUpdate struct {
	Status string
    Message string
    Code int
    Instance string
    Item int
    Total int
}

type ResponseInterpreter func(int, []byte, error)(Result, error)

func NewProcessor(user *auth.User, writer http.ResponseWriter, instances []string) *Processor {
	return &Processor{writer, instances, []string{}, user}
}

func (this *Processor) Do(method string, body interface{}, endpoint string, interpret ResponseInterpreter) error {
	var (
		completed = []string{}
		errorToReport error = nil
		total = len(this.instances)
		requests = sync.WaitGroup{}
		channels = sync.WaitGroup{}
		queue = make(chan string, 1)
		failQueue = make(chan string, 1)
	)

	if interpret == nil {
		interpret = interpretResponseDefault
	}

	this.writeUpdate(Result{"Starting", endpoint, 0}, "", 0, total)

	requests.Add(total)
	for i, inst := range this.instances {
		go func(inst string, itemNumber int) {
			defer requests.Done()

			var respBody []byte = nil
			resp, err := runBatchRequest(this.user, method, inst, endpoint, body)
			if err == nil {
				respBody, err = ioutil.ReadAll(resp.Body)
				resp.Body.Close()
			}
			result, err := interpret(resp.StatusCode, respBody, err)

			this.writeUpdate(result, inst, itemNumber, total)

			// Yield to other goroutines
			runtime.Gosched()

			if err == nil {
				queue <- inst
			} else {
				failQueue <- inst
				errorToReport = err
			}
		}(inst, i)
	}

	go func() {
		channels.Add(1)
		defer channels.Done()
		for inst := range queue {
			completed = append(completed, inst)
		}
	}()

	go func() {
		channels.Add(1)
		defer channels.Done()
		for inst := range failQueue {
			this.failures = append(this.failures, inst)
		}
	}()

	requests.Wait()
	close(queue)
	close(failQueue)
	channels.Wait()

	this.writeUpdate(Result{"Complete", endpoint, 0}, "", total, total)

	this.instances = completed
	return errorToReport
}

func (this *Processor) writeUpdate(res Result, instance string, progress, total int) {
	update := progressUpdate {
		res.Status, res.Message, res.Code, instance, progress, total,
	}

	json.NewEncoder(this.writer).Encode(update)
	if f, ok := this.writer.(http.Flusher); ok {
		f.Flush()
	}
}

func (this *Processor) FailureProcessor() *Processor {
	return NewProcessor(this.user, this.writer, this.failures);
}

func runBatchRequest(user *auth.User, method, instance, endpoint string, body interface{}) (*http.Response, error) {
	payload, _ := json.Marshal(body)
	req, err := docker.MakeDockerRequest(user, method, instance, endpoint, payload)
    if err != nil {
        return nil, err
    }

	return http.DefaultClient.Do(req)
}

func interpretResponseDefault(code int, body []byte, err error) (Result, error) {
	if err != nil {
		return Result{"Error", err.Error(), 500}, err
	}

	switch {
	case 200 <= code && code <= 299: 
		return Result{"OK", "", code}, nil

	case 300 <= code && code <= 399: 
		return Result{"Warning", "", code}, nil

	default:
		return Result{"Error", "", code}, errors.New(fmt.Sprintf("Batch request returned code %d", code))
	}
}