package ride

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/pkg/errors"
)

const limitOperations = 200000

type vm struct {
	env           RideEnvironment
	code          []byte
	ip            int
	stack         []rideType
	jmps          []int
	ref           map[uint16]point
	calls         []callLog
	numOperations int
	libVersion    int
}

func (m *vm) run() (rideType, error) {
	for m.ip < len(m.code) {
		if m.numOperations >= limitOperations {
			return nil, errors.New("limit operations exceed")
		}
		m.numOperations++

		op := m.code[m.ip]
		m.ip++
		switch op {
		case OpPop:
			_, err := m.pop()
			if err != nil {
				return nil, err
			}
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
			prop, err := m.pop()
			if err != nil {
				return nil, err //errors.Wrap(err, "no ref %d", n)
			}
			p, ok := prop.(rideString)
			if !ok {
				return nil, errors.Errorf("invalid property type '%T'", prop)
			}
			obj, err := m.pop()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get object")
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
					return nil, errors.Wrap(err, "failed to call external function")
				}
				in[i] = v
			}
			functions, err := selectFunctions(m.libVersion)
			if err != nil {
				return nil, err
			}
			provider, err := selectFunctionNameProvider(m.libVersion)
			if err != nil {
				return nil, err
			}
			fn := functions(id)
			if fn == nil {
				return nil, errors.Errorf("external function '%s' not implemented", provider(id))
			}
			res, err := fn(m.env, in...)
			m.calls = append(m.calls, callLog{
				name:   provider(id),
				args:   in,
				result: res,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "iteration %d", m.numOperations)
			}
			if isThrow(res) {
				return res, nil
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
					return v, nil
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
			if refID < 200 {
				continue
			}
			value, err := m.pop()
			if err != nil {
				return nil, errors.Wrap(err, "no value to cache")
			}
			m.push(value)
			point := m.ref[refID]
			point.value = value
			m.ref[refID] = point
		case OpClearCache:
			refID := m.uint16()
			point, ok := m.ref[refID]
			if !ok {
				return nil, errors.Errorf("OpClearCache: no ref with id %d", refID)
			}
			// Clear cache only if its not constant.
			if !point.constant() {
				point.value = nil
				m.ref[refID] = point
			}

		case OpRef:
			refID := m.uint16()
			switch {
			case refID <= 100:
				m.push(rideInt(refID))
				continue
			case refID == 101:
				m.push(rideBoolean(true))
				continue
			case refID == 102:
				m.push(rideBoolean(false))
				continue
			}
			point, ok := m.ref[refID]
			if !ok {
				return nil, errors.Errorf("reference %d not found", refID)
			}
			if point.value != nil {
				m.push(point.value)
			} else if point.fn != 0 {
				fn := predefined.getn(int(point.fn))
				rs, err := fn(m.env)
				if err != nil {
					return nil, err
				}
				m.push(rs)
			} else {
				if m.ip == int(point.position)+3 {
					return nil, errors.Errorf("infinity loop detected on iteration %d", m.numOperations)
				}
				m.jmps = append(m.jmps, m.ip)
				m.ip = int(point.position)
			}

		default:
			return nil, errors.Errorf("unknown code %#x, instruction pointer %d, at iteration %d; %s", op, m.ip, m.numOperations, hex.EncodeToString(m.code))
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
