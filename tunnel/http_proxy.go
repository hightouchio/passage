package tunnel

import (
	"bufio"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/http/httputil"
)

// handleProxyConnect reads a HTTP CONNECT request from the incoming io.ReadWriter and errors if it doesn't find that request
func handleProxyConnect(rw io.ReadWriter) error {
	// Read the initial HTTP request from the tunnel client.
	req, err := http.ReadRequest(bufio.NewReader(rw))
	if err != nil {
		return errors.Wrap(err, "could not read request")
	}

	// Assert that we receive a CONNECT request
	if req.Method != "CONNECT" {
		return errors.Errorf("expected HTTP CONNECT request, received HTTP %s", req.Method)
	}

	// Respond with 200 OK to allow client to continue sending data
	response, err := httputil.DumpResponse(&http.Response{
		Status:     "200 Connection established",
		StatusCode: http.StatusOK,
		Proto:      "HTTP",
		ProtoMajor: 1,
		ProtoMinor: 0,
	}, false)
	if err != nil {
		return errors.Wrap(err, "could not construct response")
	}
	writer := bufio.NewWriter(rw)
	if _, err := writer.Write(response); err != nil {
		return errors.Wrap(err, "could not write response")
	}
	if err := writer.Flush(); err != nil {
		return errors.Wrap(err, "could not flush response")
	}
	return nil
}
