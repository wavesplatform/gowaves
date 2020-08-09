package vm

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/op"
	"go.uber.org/zap"
	//"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
)

func fail(e error) {
	if e != nil {
		panic(e)
	}
}

type Func func(stack Context) error

//func join(b byte, bs ...byte) []byte {
//	return append([]byte{b}, bs...)
//}
//
//func str(name string) []byte {
//	l := uint16(len(name))
//	size := make([]byte, 2)
//	binary.BigEndian.PutUint16(size, l)
//	return append(size, []byte(name)...)
//}

func EvaluateExpression(code []byte, fns *Scope) (ast.Expr, error) {
	scopeVariablesAtPositions := make(map[string]int)
	stack := NewStack()
	jmps := NewJmps()

	r := NewReader(code)
	for r.HasNext() {
		n := r.Next()
		switch n {
		case op.Label:
			name := r.String()
			scopeVariablesAtPositions[name] = r.Pos() + 4 // position shift
			fns.PushVariable(name, r.Pos()+4)
			moveTo := r.I32()
			if moveTo == 0 {
				return nil, errors.Errorf("Move to zero: varname %s, curpos: %d", name, r.Pos())
			}
			r.Move(moveTo)
		case op.Call:
			name := r.String()
			err := fns.Call(name, stack)
			if err != nil {
				return nil, errors.Wrap(err, "EvaluateExpression call")
			}
			//fn, ok := fncs[name]
			//if !ok {
			//	return StackValue{}, errors.Errorf("function named '%s' not found", name)
			//}
			//err := fn(stack)
			//if err != nil {
			//	return StackValue{}, errors.Wrap(err, name)
			//}
		case op.JmpRef:
			ref := r.String()
			// first search in already computed variables
			if v, ok := fns.Calculated(ref); ok {
				stack.Push(v)
				continue
			}
			goTo, ok := scopeVariablesAtPositions[ref]
			if !ok {
				return nil, errors.Errorf("ref `%s` not found in scope", ref)
			}
			jmps.Push(r.Pos())
			r.Jmp(goTo)
		case op.StackPushL:
			stack.PushL(r.Long())

		case op.StackPushS:
			stack.Push(ast.NewString(r.String()))
		case op.Ret:
			to, ok := jmps.PopSafe()
			if ok {
				r.Jmp(to)
			} else {
				return stack.Pop(), nil
			}

		case op.StackPushTrue:
			stack.Push(ast.NewBoolean(true))
		case op.StackPushFalse:
			stack.Push(ast.NewBoolean(false))
		case op.StackPushBytes:
			stack.Push(ast.NewBytes(r.Bytes()))
		case op.JumpIfNot:
			v := stack.Pop()
			b, ok := v.(*ast.BooleanExpr)
			if !ok {
				return nil, errors.Errorf("expected argument for if (JumpIfNot) to be `*ast.BooleanExpr`, found %T", v)
			}
			shift := r.I32()
			if !b.Value {
				//jmps.Push(r.Pos())
				r.Jmp(int(shift))
			}
		case op.Jmp:
			i := r.I32()
			r.Move(i)
		default:
			return nil, errors.Errorf("unknown opcode %d at pos %d", n, r.Pos())
		}
	}

	return stack.Pop(), nil
}

func EvaluateExpressionAsBoolean(code []byte, scope *Scope) (bool, error) {
	rs, err := EvaluateExpression(code, scope)
	if err != nil {
		return false, err
	}
	if rs == nil {
		zap.S().Debugf("EvaluateExpressionAsBoolean ret is null: %+v", code)
		return false, errors.New("script result is nil")
	}
	return rs.(*ast.BooleanExpr).Value, nil
}
