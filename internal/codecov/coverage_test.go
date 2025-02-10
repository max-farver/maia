package codecov

import (
	"fmt"
	"testing"
)

func TestGetDiff(t *testing.T) {
	diff, err := GetDiff()
	if err != nil {
		t.Fatalf("error getting diff: %v", err)
	}

	coverage, err := GetCoverage(diff, "output.txt")
	if err != nil {
		t.Fatalf("error getting coverage: %v", err)
	}

	fmt.Println(coverage)
}
