package ride

func newHeight(env Environment) rideType {
	if env == nil {
		return rideUnit{}
	}
	return env.height()
}

func newTx(env Environment) rideType {
	if env == nil {
		return rideUnit{}
	}
	tx := env.transaction()
	if tx == nil {
		return rideUnit{}
	}
	return tx
}

func newLastBlock(env Environment) rideType {
	if env == nil {
		return rideUnit{}
	}
	b := env.block()
	if b == nil {
		return rideUnit{}
	}
	return b
}

func newThis(env Environment) rideType {
	if env == nil {
		return rideUnit{}
	}
	this := env.this()
	if this == nil {
		return rideUnit{}
	}
	return this
}

func newInvocation(env Environment) rideType {
	if env == nil {
		return rideUnit{}
	}
	inv := env.invocation()
	if inv == nil {
		return rideUnit{}
	}
	return inv
}

func newUnit(Environment) rideType {
	return rideUnit{}
}

func newNil(Environment) rideType {
	return rideList(nil)
}
