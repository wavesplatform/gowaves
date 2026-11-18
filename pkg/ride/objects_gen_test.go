package ride //nolint:testpackage // White-box tests of the unexported generated objects.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generatedObjects returns a zero value of every object type generated into objects.gen.go.
func generatedObjects() []rideType {
	return []rideType{
		rideAssetV3{},
		rideAssetV4{},
		rideAssetPair{},
		rideAttachedPayment{},
		rideBalanceDetails{},
		rideBinaryEntry{},
		rideBlockInfoV3{},
		rideBlockInfoV4{},
		rideBlockInfoV7{},
		rideBooleanEntry{},
		rideBurn{},
		rideDataEntry{},
		rideDeleteEntry{},
		rideIntegerEntry{},
		rideInvocationV3{},
		rideInvocationV4{},
		rideInvocationV5{},
		rideIssue{},
		rideLeaseCancel{},
		rideLease{},
		rideOrderV1{},
		rideOrderV8{},
		rideReissue{},
		rideScriptResult{},
		rideScriptTransfer{},
		rideSponsorFee{},
		rideStringEntry{},
		rideTransfer{},
		rideTransferSet{},
		rideWriteSet{},
		rideBurnTransaction{},
		rideCreateAliasTransaction{},
		rideDataTransaction{},
		rideExchangeTransaction{},
		rideGenesisTransaction{},
		rideInvokeExpressionTransaction{},
		rideInvokeScriptTransactionV3{},
		rideInvokeScriptTransactionV4{},
		rideIssueTransaction{},
		rideLeaseCancelTransaction{},
		rideLeaseTransaction{},
		rideMassTransferTransaction{},
		ridePaymentTransaction{},
		rideReissueTransaction{},
		rideSetAssetScriptTransaction{},
		rideSetScriptTransaction{},
		rideSponsorFeeTransaction{},
		rideTransferTransaction{},
		rideUpdateAssetInfoTransaction{},
		rideCommitToGenerationTransaction{},
	}
}

// generatedObjectTypeNames returns the names of the types that have the generated get method in objects.gen.go.
func generatedObjectTypeNames(t *testing.T) []string {
	f, err := parser.ParseFile(token.NewFileSet(), "objects.gen.go", nil, parser.SkipObjectResolution)
	require.NoError(t, err)
	var names []string
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || fd.Name.Name != "get" {
			continue
		}
		if id, isIdent := fd.Recv.List[0].Type.(*ast.Ident); isIdent {
			names = append(names, id.Name)
		}
	}
	return names
}

func TestGeneratedObjectsCoverage(t *testing.T) {
	covered := make(map[string]bool)
	for _, obj := range generatedObjects() {
		covered[strings.TrimPrefix(fmt.Sprintf("%T", obj), "ride.")] = true
	}
	names := generatedObjectTypeNames(t)
	require.NotEmpty(t, names)
	for _, name := range names {
		assert.Truef(t, covered[name], "generated object type %s is missing in generatedObjects", name)
	}
	assert.Len(t, covered, len(names))
}

func TestGeneratedObjectsGetInstance(t *testing.T) {
	for _, obj := range generatedObjects() {
		t.Run(fmt.Sprintf("%T", obj), func(t *testing.T) {
			v, err := obj.get(instanceField)
			require.NoError(t, err)
			assert.Equal(t, rideString(obj.instanceOf()), v)
			allocs := testing.AllocsPerRun(100, func() {
				_, _ = obj.get(instanceField)
			})
			assert.Zero(t, allocs)
		})
	}
}

func TestGeneratedObjectsGetUnknownProperty(t *testing.T) {
	for _, obj := range generatedObjects() {
		t.Run(fmt.Sprintf("%T", obj), func(t *testing.T) {
			for _, prop := range []string{"unknownProperty", "", "$Instance", "instance"} {
				v, err := obj.get(prop)
				assert.Nil(t, v)
				assert.EqualError(t, err, fmt.Sprintf("type '%s' has no property '%s'", obj.instanceOf(), prop))
			}
		})
	}
}
