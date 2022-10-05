package tunnel

import (
	"bufio"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net"
	"sync"
	"testing"
)

func writeLine(contents string) func(tc tunneledConn, t *testing.T) {
	return func(tc tunneledConn, t *testing.T) {
		logrus.WithField("contents", contents).Debug("write line")
		if _, err := tc.write.WriteString(contents + "\n"); err != nil {
			t.Fatal(errors.Wrapf(err, "could not write line %s", contents))
		}
		if err := tc.write.Flush(); err != nil {
			t.Fatal(errors.Wrap(err, "could not flush"))
		}
	}
}

// read one line from the connection and assert that the string is equal
func readAndAssertLine(expected string) func(tc tunneledConn, t *testing.T) {
	return func(tc tunneledConn, t *testing.T) {
		logrus.WithField("contents", expected).Debug("expect line")
		ok := tc.read.Scan()
		if !ok {
			t.Fatal("nothing more to scan")
		}

		assert.Equal(t, expected, tc.read.Text())
	}
}

type testInstruction func(tc tunneledConn, t *testing.T)

type tunneledConn struct {
	conn  net.Conn
	read  *bufio.Scanner
	write *bufio.Writer
}

func newTunneledConn(c net.Conn) tunneledConn {
	scanner := bufio.NewScanner(c)
	writer := bufio.NewWriter(c)

	return tunneledConn{
		conn:  c,
		read:  scanner,
		write: writer,
	}
}

func (tc tunneledConn) runAssertions(t *testing.T, wg *sync.WaitGroup, instructions []testInstruction) {
	defer tc.conn.Close()
	defer wg.Done()

	// run instructions
	for _, inst := range instructions {
		inst(tc, t)
		if t.Failed() {
			return
		}
	}
}
