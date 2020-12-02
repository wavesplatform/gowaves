package ride

import (
	"encoding/binary"

	//im "github.com/frozen/immutable_map"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type vm struct {
	env          RideEnvironment
	code         []byte
	ip           int
	constants    []rideType
	functions    func(int) rideFunction
	globals      func(int) rideConstructor
	stack        []rideType
	functionName func(int) string
	jmps         []int
	ref          map[uint16]point
	cache        bool
}

func (m *vm) run() (RideResult, error) {
	numOperations := 0
	limitOperations := 20000
	//if m.stack != nil {
	//	m.stack = m.stack[0:0]
	//}

	for m.ip < len(m.code) {
		if numOperations >= limitOperations {
			return nil, errors.New("limit operations exceed")
		}
		numOperations++

		op := m.code[m.ip]
		m.ip++
		switch op {
		case OpJump:
			pos := m.arg16()
			m.jmps = append(m.jmps, m.ip)
			m.ip = pos

		case OpJumpIfFalse:
			posTrue := m.arg16()
			posFalse := m.arg16()
			posNext := m.arg16()
			m.jmps = append(m.jmps, posNext)

			val, err := m.pop()
			if err != nil {
				return nil, errors.Wrap(err, "OpJumpIfFalse")
			}
			v, ok := val.(rideBoolean)
			if !ok {
				return nil, errors.Errorf("not a boolean value '%v' of type '%T'", m.current(), m.current())
			}
			if v {
				m.ip = posTrue
			} else {
				m.ip = posFalse
			}
		case OpProperty:
			obj, err := m.pop()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get object")
			}
			prop := m.ref[m.uint16()].value
			p, ok := prop.(rideString)
			if !ok {
				return nil, errors.Errorf("invalid property name type '%s'", prop.instanceOf())
			}
			v, err := obj.get(string(p))
			if err != nil {
				return nil, errors.Wrap(err, "vm OpProperty")
			}
			m.push(v)
		case OpCall:
			pos := m.arg16()
			m.jmps = append(m.jmps, m.ip)
			m.ip = pos

		case OpExternalCall:
			// Before calling external function all parameters must be evaluated and placed on stack
			id := m.arg16()
			cnt := m.arg16()
			in := make([]rideType, cnt) // Prepare input parameters for external call
			for i := cnt - 1; i >= 0; i-- {
				v, err := m.pop()
				if err != nil {
					return nil, errors.Wrapf(err, "failed to call external function '%s'", m.functionName(id))
				}
				in[i] = v
			}
			fn := m.functions(id)
			if fn == nil {
				return nil, errors.Errorf("external function '%s' not implemented", m.functionName(id))
			}
			res, err := fn(m.env, in...)
			if err != nil {
				return nil, errors.Wrapf(err, "iteration %d", numOperations)
			}
			if isThrow(res) {
				return nil, errors.Errorf("terminated execution by throw with message %q", res)
			}
			m.push(res)
		case OpReturn:
			l := len(m.jmps)
			if l == 0 {
				if len(m.stack) > 0 {
					v, err := m.pop()
					if err != nil {
						return nil, errors.Wrap(err, "failed to get result value")
					}
					switch tv := v.(type) {
					case rideBoolean:
						return ScriptResult{res: bool(tv), operations: numOperations}, nil
					case rideObject:
						actions, err := objectToActions(m.env, tv)
						if err != nil {
							return nil, errors.Wrap(err, "failed to convert evaluation result")
						}
						return DAppResult{res: true, actions: actions, msg: ""}, nil
					case rideList:
						actions := make([]proto.ScriptAction, len(tv))
						for i, item := range tv {
							a, err := convertToAction(m.env, item)
							if err != nil {
								return nil, errors.Wrap(err, "failed to convert evaluation result")
							}
							actions[i] = a
						}
						return DAppResult{res: true, actions: actions}, nil
					default:
						return nil, errors.Errorf("unexpected result value '%v' of type '%T'", v, v)
					}
				}
				return nil, errors.New("no result after script execution")
			}
			m.ip, m.jmps = m.jmps[l-1], m.jmps[:l-1]

		case OpSetArg:
			from := m.uint16()
			to := m.uint16()
			// for debug purpose
			x := m.ref[from]
			_ = x
			m.ref[to] = m.ref[from]
		case OpCache:
			refID := m.uint16()
			value, err := m.pop()
			if err != nil {
				return nil, errors.Wrap(err, "no value to cache")
			}
			m.ref[refID] = point{
				value: value,
			}

		case OpRef:
			refID := m.uint16()
			point, ok := m.ref[refID]
			if !ok {
				return nil, errors.Errorf("reference %d not found", refID)
			}
			if point.value != nil {
				m.push(point.value)
			} else if point.fn != nil {
				rs, err := point.fn(m.env)
				if err != nil {
					return nil, err
				}
				m.push(rs)
			} else {
				m.jmps = append(m.jmps, m.ip)
				m.ip = int(point.position)
			}

		default:
			return nil, errors.Errorf("unknown code %#x, at iteration %d", op, numOperations)
		}
	}
	return nil, errors.New("broken code")
}

func (m *vm) push(v rideType) constid {
	m.stack = append(m.stack, v)
	return uint16(len(m.stack) - 1)
}

func (m *vm) pop() (rideType, error) {
	if len(m.stack) == 0 {
		return nil, errors.New("empty callStack")
	}
	value := m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	return value, nil
}

func (m *vm) current() rideType {
	return m.stack[len(m.stack)-1]
}

func (m *vm) arg16() int {
	//TODO: add check
	res := binary.BigEndian.Uint16(m.code[m.ip : m.ip+2])
	m.ip += 2
	return int(res)
}

func (m *vm) uint16() uint16 {
	//TODO: add check
	res := binary.BigEndian.Uint16(m.code[m.ip : m.ip+2])
	m.ip += 2
	return res
}

func (m *vm) constant() rideType {
	//TODO: add check
	return m.constants[m.arg16()]
}
