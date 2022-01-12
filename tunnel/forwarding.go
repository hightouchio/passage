package tunnel

import "io"

// BidirectionalPipeline passes bytes bidirectionally from io.ReadWriters a and b, and records the number of bytes written to each.
type BidirectionalPipeline struct {
	a, b               io.ReadWriter
	writtenA, writtenB int64
}

func NewBidirectionalPipeline(a, b io.ReadWriter) *BidirectionalPipeline {
	return &BidirectionalPipeline{a: a, b: b}
}

// Run starts the bidirectional copying of bytes, and blocks until completion.
func (p *BidirectionalPipeline) Run() error {
	// Buffered error channel to allow both sides to send an error without blocking and leaking goroutines.
	cerr := make(chan error, 1)
	// Copy data bidirectionally.
	go func() {
		cerr <- copyWithCounter(p.a, p.b, &p.writtenB)
	}()
	go func() {
		cerr <- copyWithCounter(p.b, p.a, &p.writtenA)
	}()

	// Wait for either side A or B to close, and return err
	return <-cerr
}

// copyWithCounter performs an io.Copy from src to dst, and keeps track of the number of bytes written by writing to the *written pointer.
func copyWithCounter(src io.Reader, dst io.Writer, written *int64) error {
	count, err := io.Copy(io.MultiWriter(dst, CounterWriter{written}), src)
	*written = count
	return err
}

// CounterWriter is a no-op Writer that records how many bytes have been written to it
type CounterWriter struct {
	written *int64
}

// Write does nothing with the input byte slice but records the length
func (b CounterWriter) Write(p []byte) (n int, err error) {
	count := len(p)
	*b.written += int64(count)
	return count, nil
}
