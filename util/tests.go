package util

import (
    "testing"
)

func Chkerr(t *testing.T, err error) {
    if err != nil {
        t.Errorf("Error encountered: %s", err)
    }
}
