package tunnel

import (
	"bufio"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/http/httputil"
)

// handleHttpProxy proxies HTTP requests by passing bytes directly to the upstream, or by handling an initial CONNECT request
func handleHttpProxy(rw io.ReadWriter, upstream io.Writer) error {
	// Read the initial HTTP request from the tunnel client.
	req, err := http.ReadRequest(bufio.NewReader(rw))
	if err != nil {
		return errors.Wrap(err, "could not read request")
	}

	// If we didn't receive a CONNECT request, just pass the bytes on to the upstream like before
	if req.Method != http.MethodConnect {
		requestBytes, err := httputil.DumpRequest(req, true)
		if err != nil {
			return errors.Wrap(err, "could not dump initial request bytes")
		}
		if _, err := upstream.Write(requestBytes); err != nil {
			return errors.Wrap(err, "could not proxy initial request to upstream")
		}

		// If we've successfully written the original request to the upstream, we can stop proxying.
		return nil
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
