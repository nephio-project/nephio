package fn

import (
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	tst "github.com/nephio-project/nephio/krm-functions/lib/test"
)

const TestGoldenPath = "testdata/golden"
const TestFailurePath = "testdata/failure"

func TestGolden(t *testing.T) {
	fnRunner := fn.ResourceListProcessorFunc(Run)

	//// This golden test expects each sub-directory of `testdata` can has its input resources (in `resources.yaml`)
	//// be modified to the output resources (in `_expected_error.txt`).
	tst.RunGoldenTests(t, TestGoldenPath, fnRunner)
}

func TestFailureCases(t *testing.T) {
	fnRunner := fn.ResourceListProcessorFunc(Run)

	tst.RunGoldenTests(t, TestFailurePath, fnRunner)
}
