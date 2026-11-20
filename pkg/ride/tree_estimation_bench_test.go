package ride //nolint:testpackage // White-box tests of the unexported estimator internals.

import (
	"encoding/base64"
	stderrs "errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	ridec "github.com/wavesplatform/gowaves/pkg/ride/compiler"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
)

// estimatorCorpusScript is a parsed or compiled script used by the estimator benchmarks and the golden test.
type estimatorCorpusScript struct {
	name     string
	tree     *ast.Tree
	versions []int // Estimator versions applicable to the script.
}

var allEstimatorVersions = []int{1, 2, 3, 4} //nolint:gochecknoglobals // Read-only list of the estimator versions.

// loadEstimatorCorpus builds the estimator corpus. The order of the scripts is stable, because the golden
// test relies on it. The corpus consists of:
//   - the base64 scripts of TestEstimatorCommon;
//   - the base64 scripts of the error and compatibility tests;
//   - the Ride sources from testData and testData/estimator.
func loadEstimatorCorpus(tb testing.TB) []estimatorCorpusScript {
	tb.Helper()
	corpus := make([]estimatorCorpusScript, 0, len(estimatorCommonTestCases)+16)
	addBase64 := func(name, src string, versions []int) {
		tb.Helper()
		corpus = append(corpus, estimatorCorpusScript{name: name, tree: mustParseBase64Tree(tb, src), versions: versions})
	}
	for i, tc := range estimatorCommonTestCases {
		addBase64(fmt.Sprintf("common_%03d", i), tc.source, allEstimatorVersions)
	}
	// Estimation time of the cost overflow script grows exponentially on V1 and V2, they never finish.
	addBase64("cost_overflow", costOverflowScript, []int{3, 4})
	addBase64("invocation_in_expression", invocationInExpressionScript, allEstimatorVersions)
	addBase64("invocation_in_dapp", invocationInDAppScript, allEstimatorVersions)
	addBase64("invocation_in_verifier", invocationInVerifierScript, allEstimatorVersions)
	addBase64("ride_v9_scala_compatibility", rideV9ScalaCompatibilityScript, allEstimatorVersions)
	for _, dir := range []struct{ path, prefix string }{
		{path: "testData", prefix: "testdata"}, // The directory name is case-sensitive on some file systems.
		{path: filepath.Join("testData", "estimator"), prefix: "testdata_estimator"},
	} {
		files, err := filepath.Glob(filepath.Join(dir.path, "*.ride"))
		require.NoError(tb, err)
		require.NotEmpty(tb, files, dir.path)
		slices.Sort(files)
		for _, fn := range files {
			src, rErr := os.ReadFile(fn)
			require.NoError(tb, rErr)
			tree, errs := ridec.CompileToTree(string(src))
			require.NoError(tb, stderrs.Join(errs...), fn)
			name := dir.prefix + "_" + strings.TrimSuffix(filepath.Base(fn), ".ride")
			corpus = append(corpus, estimatorCorpusScript{name: name, tree: tree, versions: allEstimatorVersions})
		}
	}
	return corpus
}

func mustParseBase64Tree(tb testing.TB, src string) *ast.Tree {
	tb.Helper()
	script, err := base64.StdEncoding.DecodeString(src)
	require.NoError(tb, err)
	tree, err := serialization.Parse(script)
	require.NoError(tb, err)
	require.NotNil(tb, tree)
	return tree
}

// estimatorBenchmarkResult keeps the benchmark results reachable, so the compiler can't drop the estimation.
var estimatorBenchmarkResult TreeEstimation //nolint:gochecknoglobals // Benchmark sink.

type estimateFunc func(tree *ast.Tree) (TreeEstimation, error)

func estimateV1(tree *ast.Tree) (TreeEstimation, error) {
	te, err := newTreeEstimatorV1(tree)
	if err != nil {
		return TreeEstimation{}, err
	}
	e, v, f, err := te.estimate()
	return TreeEstimation{Estimation: e, Verifier: v, Functions: f}, err
}

func estimateV2(tree *ast.Tree) (TreeEstimation, error) {
	te, err := newTreeEstimatorV2(tree)
	if err != nil {
		return TreeEstimation{}, err
	}
	e, v, f, err := te.estimate()
	return TreeEstimation{Estimation: e, Verifier: v, Functions: f}, err
}

func estimateV3(tree *ast.Tree) (TreeEstimation, error) {
	te, err := newTreeEstimatorV3(tree)
	if err != nil {
		return TreeEstimation{}, err
	}
	e, v, f, err := te.estimate()
	return TreeEstimation{Estimation: e, Verifier: v, Functions: f}, err
}

func estimateV4(tree *ast.Tree) (TreeEstimation, error) {
	te, err := newTreeEstimatorV4(tree)
	if err != nil {
		return TreeEstimation{}, err
	}
	e, v, f, err := te.estimate()
	return TreeEstimation{Estimation: e, Verifier: v, Functions: f}, err
}

func benchmarkEstimator(b *testing.B, version int, estimate estimateFunc) {
	for _, s := range loadEstimatorCorpus(b) {
		if !slices.Contains(s.versions, version) {
			continue
		}
		if _, ok := treeEstimationV2Override(s.tree); ok && version == 2 {
			continue // The V2 estimator overflows the stack on these scripts, EstimateTree returns hard-coded results.
		}
		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				estimatorBenchmarkResult, _ = estimate(s.tree) // Errors are expected for some scripts.
			}
		})
	}
}

func BenchmarkEstimatorV1(b *testing.B) { benchmarkEstimator(b, 1, estimateV1) }

func BenchmarkEstimatorV2(b *testing.B) { benchmarkEstimator(b, 2, estimateV2) }

func BenchmarkEstimatorV3(b *testing.B) { benchmarkEstimator(b, 3, estimateV3) }

func BenchmarkEstimatorV4(b *testing.B) { benchmarkEstimator(b, 4, estimateV4) }

func BenchmarkEstimateTree(b *testing.B) {
	corpus := loadEstimatorCorpus(b)
	for v := 1; v <= 4; v++ {
		b.Run(fmt.Sprintf("V%d", v), func(b *testing.B) {
			for _, s := range corpus {
				if !slices.Contains(s.versions, v) {
					continue
				}
				b.Run(s.name, func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						estimatorBenchmarkResult, _ = EstimateTree(s.tree, v) // Errors are expected for some scripts.
					}
				})
			}
		})
	}
}
