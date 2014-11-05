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
        "testing"

        "net/http"
        "net/http/httptest"
        "bytes"

        ".." // Lighthouse
)

// Helper for GetRequestBody tests
func makeRequestAndGetBody(body string) *RequestBody {
    byteBody := []byte(body)
    req, _ := http.NewRequest("", "", bytes.NewBuffer(byteBody))
    return GetRequestBody(req)
}

/*
    Tests body extraction for body-less requests
    Purpose: GET and DELETE requests
*/
func Test_GetRequestBody_Null(t *testing.T) {
    req, _ := http.NewRequest("", "", nil)

    if GetRequestBody(req) != nil {
        t.Error("GetResponseBody should return nil on nil body")
    }
}

/*
    Tests body extraction for bodied requests
    Purpose: [most] POST and PUT requests
*/
func Test_GetRequestBody_Normal(t *testing.T) {
    outBody := makeRequestAndGetBody(`{"Payload":"TestPayload"}`)

    if outBody == nil {
        t.Error("GetResponseBody should not return nil on non-nil body")
    }
    if outBody.Payload != "TestPayload" {
        t.Error("GetResponseBody did not extract Payload correctly")
    }
}

/*
    Tests body extraction for requests with extra field.
    Purpose: To add niche fields to requests without bogging
        down the RequestBody type.
*/
func Test_GetRequestBody_ExtraPayload(t *testing.T) {
    outBody := makeRequestAndGetBody(`{"Payload":"TestPayload","Extra":"ExtraField"}`)

    if outBody == nil {
        t.Error("GetResponseBody should not return nil on non-nil body")
    }
    if outBody.Payload != "TestPayload" {
        t.Error("GetResponseBody did not extract Payload correctly with an extra field")
    }
}

/*
    Tests body extraction for requests without a Payload.
    Purpose: To add niche fields to requests without bogging
        down the RequestBody type.
*/
func Test_GetRequestBody_NoPayload(t *testing.T) {
    outBody := makeRequestAndGetBody(`{"NotAPayload":"TotallyNotAPayload"}`)

    if outBody == nil {
        t.Error("GetResponseBody should not return nil on non-nil body")
    }
    if outBody.Payload != "" {
        t.Error("GetResponseBody did not extract Payload correctly with an extra field")
    }
}
