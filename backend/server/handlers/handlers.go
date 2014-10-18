package handlers

import (
    "net/http"
    "regexp"
)

/*
    ---------------------------------------------------------------------
      THE CUSTOM HANDLER INTERFACE
    ---------------------------------------------------------------------

      A custom handler will:
    ---------------------------------------------------------------------
      - retrieve all data it requires from the request given to it

      - perform any operations neccessary to rollback changes it has
        made for a perticular request

      - perform its own error detection and recovery

      - rollback and return a non-nil *HandlerError if a failure occurs

      - return nil upon successful completion

      A custom handler may safely assume:
    ---------------------------------------------------------------------
      - r.ParseForm() will have been called prior to calling the handler

      - it will not be asked to rollback if it has not been called
        already for a particular request

      - it will not be asked to rollback if it caused an unrecoverable
        error for a particular request (i.e. initiated a rollback)

      - its error message will be returned to the client during failure
*/

type HandlerError struct {
    StatusCode int
    Cause string
    Message string
}

type customHandlerFunc func(r *http.Request, rollback bool) (*HandlerError)

/*
    Searches for any applicable custom handlers for the given request.

    RETURN: A list of custom handlers which have run (for rollback)
            an non-nil *HandlerError on failure, nil otherwise
*/
func RunCustomHandlers(r *http.Request, url string) ([]customHandlerFunc, *HandlerError) {

    customHandlers := map[*regexp.Regexp]customHandlerFunc{
        //regexp.MustCompile("example"): ExampleHandler,
    }

    runHandlers := []customHandlerFunc{}

    for exp, handler := range customHandlers {
        if exp.MatchString(url) {
            if res := handler(r, false); res != nil {
                return runHandlers, res
            }
            runHandlers = append(runHandlers, handler)
        }
    }

    return runHandlers, nil
}
