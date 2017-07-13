package bugsnack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"

	"github.com/fromatob/bugsnack/error"
	"github.com/fromatob/bugsnack/hashstruct"
	"github.com/fromatob/bugsnack/internal/stack"
)

const clientVersion = "0.0.2"

// BugsnagReporter is an implementation of ErrorReporter that fires to
// BugSnag
type BugsnagReporter struct {
	Doer         Doer
	APIKey       string
	ReleaseStage string

	Backup ErrorReporter
}
type bugsnagMetadata struct {
	errorClass    string
	context       string
	groupingHash  string
	severity      string
	eventMetadata *hashstruct.Hash
}

func (er *BugsnagReporter) Report(ctx context.Context, newErr *error.Error, metadata *bugsnagMetadata) {
	payload := er.newPayload(newErr, metadata)

	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(payload)
	if err != nil {
		er.Backup.Report(ctx, error.New(err), &bugsnagMetadata{})
		return
	}

	req, err := http.NewRequest(http.MethodPost, "https://notify.bugsnag.com", &b)
	if err != nil {
		er.Backup.Report(ctx, error.New(err), &bugsnagMetadata{})
		return
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")

	resp, err := er.Doer.Do(req)
	if err != nil {
		er.Backup.Report(ctx, error.New(err), &bugsnagMetadata{})
		return
	}
	defer func() {
		_, err = io.Copy(ioutil.Discard, io.LimitReader(resp.Body, 1024))
		if err != nil {
			er.Backup.Report(ctx, error.New(err), &bugsnagMetadata{})
		}
		err = resp.Body.Close()
		if err != nil {
			er.Backup.Report(ctx, error.New(err), &bugsnagMetadata{})
		}
	}()

	if resp.StatusCode != http.StatusOK {
		er.Backup.Report(ctx, error.New("could not report to bugsnag"), &bugsnagMetadata{})
		return
	}
}

func (er *BugsnagReporter) newPayload(err *error.Error, metadata *bugsnagMetadata) *hashstruct.Hash {
	populateMetadata(metadata, err)

	host, _ := os.Hostname()
	return &hashstruct.Hash{
		"apiKey": er.APIKey,

		"notifier": hashstruct.Hash{
			"name":    "Bugsnack/Bugsnag",
			"url":     "https://github.com/fromatob/bugsnack",
			"version": clientVersion,
		},

		"events": []hashstruct.Hash{
			{
				"PayloadVersion": "2",
				"exceptions": []hashstruct.Hash{
					{
						"errorClass": metadata.errorClass,
						"message":    err.Error(),
						"stacktrace": formatStack(err.Stacktrace),
					},
				},
				"severity": metadata.severity,
				"app": hashstruct.Hash{
					"releaseStage": er.ReleaseStage,
				},
				"device": hashstruct.Hash{
					"hostname": host,
				},
				"context":      metadata.context,
				"groupingHash": metadata.groupingHash,
				"metaData":     metadata.eventMetadata,
			},
		},
	}
}

func populateMetadata(metadata *bugsnagMetadata, err *error.Error) {
	if metadata.errorClass == "" {
		metadata.errorClass = reflect.TypeOf(err).String()
	}
	if metadata.severity == "" {
		metadata.severity = "error"
	}
}

func formatStack(s stack.CallStack) []hashstruct.Hash {
	var o []hashstruct.Hash

	for _, f := range s {
		line, _ := strconv.Atoi(fmt.Sprintf("%d", f))
		o = append(o, hashstruct.Hash{
			"method":     fmt.Sprintf("%n", f),
			"file":       fmt.Sprintf("%s", f),
			"lineNumber": line,
		})
	}

	return o
}
