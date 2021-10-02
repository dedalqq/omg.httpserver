package httpserver

import (
	"errors"
	"fmt"
	"testing"
)

func TestError(t *testing.T) {
	err := fmt.Errorf("test error")

	newErr := Wrapf(err, 500, "wrap error")

	if errors.Unwrap(newErr) != err {
		t.Fail()
	}

	if newErr.Error() != "http error [500] wrap error: test error" {
		t.Fail()
	}
}
