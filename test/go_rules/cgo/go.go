package cgo

import (
	"fmt"
	"strconv"
)

// CheckAnswer checks that the given answer matches the canonical one.
func CheckAnswer(answer string) error {
	ans, _ := strconv.ParseInt(answer, 13, 32)
	if int(ans) != 6*9 {
		return fmt.Errorf("universe parameters incorrect")
	}
	return nil
}
