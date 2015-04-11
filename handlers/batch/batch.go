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
	"bytes"
	"errors"
	"runtime"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

type Processor struct {
	writer http.ResponseWriter
	instances []string
	failures []string
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

type ResponseInterpreter func(resp *http.Response, err error)(Result, error)

func NewProcessor(writer http.ResponseWriter, instances []string) *Processor {
	return &Processor{writer, instances, []string{}}
}

func (this *Processor) Do(method string, body interface{}, endpoint string, interpret ResponseInterpreter) error {
	var completed []string
	var errorToReport error = nil
	var total int = len(this.instances)
	var requests, channels sync.WaitGroup
	var queue chan string = make(chan string, 1)
	var failQueue chan string = make(chan string, 1)

	if interpret == nil {
		interpret = interpretResponseDefault
	}

	this.writeUpdate(Result{"Starting", endpoint, 0}, "", 0, total)

	requests.Add(total)
	for i, inst := range this.instances {
		go func(inst string, itemNumber int) {
			defer requests.Done()

			dest := fmt.Sprintf("http://%s/%s", inst, endpoint)
			resp, err := runBatchRequest(method, dest, body)
			ioutil.ReadAll(resp.Body)
			result, err := interpret(resp, err)

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

	jsonBody, _ := json.Marshal(update)
	this.writer.Write(jsonBody)
}

func (this *Processor) FailureProcessor() *Processor {
	return NewProcessor(this.writer, this.failures);
}

func runBatchRequest(method, dest string, body interface{}) (*http.Response, error) {
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest(method, dest, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(req)
}

func interpretResponseDefault(resp *http.Response, err error) (Result, error) {
	if err != nil {
		return Result{"Error", err.Error(), 500}, err
	}

	code := resp.StatusCode

	switch {
	case 200 <= code && code <= 299: 
		return Result{"OK", "", code}, nil

	case 300 <= code && code <= 399: 
		return Result{"Warning", "", code}, nil

	default:
		return Result{"Error", "", code}, errors.New(fmt.Sprintf("Batch request returned code %d", code))
	}
}