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
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"sync"

	"github.com/lighthouse/lighthouse/auth"
	"github.com/lighthouse/lighthouse/handlers/docker"
)

type Processor struct {
	writer    http.ResponseWriter
	instances []string
	failures  []string
	user      *auth.User
	writeLock sync.Mutex
}

type Result struct {
	Status  string
	Message string
	Code    int
}

type progressUpdate struct {
	Status   string
	Method   string
	Endpoint string
	Message  string
	Code     int
	Instance string
	Item     int
	Total    int
}

type ResponseInterpreter func(int, io.Reader, error) (Result, error)

func NewProcessor(user *auth.User, writer http.ResponseWriter, instances []string) *Processor {
	return &Processor{writer, instances, []string{}, user, sync.Mutex{}}
}

func (this *Processor) Do(action, method string, body interface{}, endpoint string, interpret ResponseInterpreter) error {
	var (
		completed           = []string{}
		errorToReport error = nil
		total               = len(this.instances)
		requests            = sync.WaitGroup{}
		channels            = sync.WaitGroup{}
		queue               = make(chan string, 1)
		failQueue           = make(chan string, 1)
	)

	if interpret == nil {
		interpret = interpretResponseDefault
	}

	this.writeUpdate(Result{"Starting", action, 0}, method, endpoint, "", 0, total)

	requests.Add(total)
	for i, inst := range this.instances {
		go func(inst string, itemNumber int) {
			defer requests.Done()

			resp, err := runBatchRequest(this.user, method, inst, endpoint, body)
			result, err := interpret(resp.StatusCode, resp.Body, err)

			// Make sure the response is complete before ending
			if err == nil {
				ioutil.ReadAll(resp.Body)
				resp.Body.Close()
			}

			this.writeUpdate(result, method, endpoint, inst, itemNumber, total)

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

	this.writeUpdate(Result{"Complete", action, 0}, method, endpoint, "", total, total)

	this.instances = completed
	return errorToReport
}

func (this *Processor) FailureProcessor() *Processor {
	return NewProcessor(this.user, this.writer, this.failures)
}

func Finalize(w http.ResponseWriter) {
	json.NewEncoder(w).Encode(progressUpdate{Status: "Finalized"})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (this *Processor) writeUpdate(res Result, method, endpoint, instance string, progress, total int) {
	update := progressUpdate{
		Status:   res.Status,
		Method:   method,
		Endpoint: endpoint,
		Message:  res.Message,
		Code:     res.Code,
		Instance: instance,
		Item:     progress,
		Total:    total,
	}

	this.writeLock.Lock()
	defer this.writeLock.Unlock()

	json.NewEncoder(this.writer).Encode(update)
	if f, ok := this.writer.(http.Flusher); ok {
		f.Flush()
	}
}

func runBatchRequest(user *auth.User, method, instance, endpoint string, body interface{}) (*http.Response, error) {
	payload, _ := json.Marshal(body)
	req, err := docker.MakeDockerRequest(user, method, instance, endpoint, payload)
	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(req)
}

func interpretResponseDefault(code int, body io.Reader, err error) (Result, error) {
	if err != nil {
		return Result{"Error", err.Error(), 500}, err
	}

	msg := make([]byte, 83)
	cnt, err := io.ReadFull(body, msg)

	if cnt > 80 {
		msg = append(msg[:80], []byte("...")...)
	} else {
		msg = msg[:cnt]
	}

	switch {
	case 200 <= code && code <= 299:
		return Result{"OK", string(msg), code}, nil

	case 300 <= code && code <= 399:
		return Result{"Warning", string(msg), code}, nil

	default:
		return Result{"Error", string(msg), code}, errors.New(string(msg))
	}
}
