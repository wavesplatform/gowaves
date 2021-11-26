package ride

func newHeight(env environment) rideType {
	if env == nil {
		return rideUnit{}
	}
	return env.height()
}

func newTx(env environment) rideType {
	if env == nil {
		return rideUnit{}
	}
	tx := env.transaction()
	if tx == nil {
		return rideUnit{}
	}
	return tx
}

func newLastBlock(env environment) rideType {
	if env == nil {
		return rideUnit{}
	}
	b := env.block()
	if b == nil {
		return rideUnit{}
	}
	return b
}

func newThis(env environment) rideType {
	if env == nil {
		return rideUnit{}
	}
	this := env.this()
	if this == nil {
		return rideUnit{}
	}
	return this
}

func newInvocation(env environment) rideType {
	if env == nil {
		return rideUnit{}
	}
	inv := env.invocation()
	if inv == nil {
		return rideUnit{}
	}
	return inv
}

func newUnit(environment) rideType {
	return rideUnit{}
}

func newNil(environment) rideType {
	return rideList(nil)
}
