package ride

func newHeight(env RideEnvironment) rideType {
	if env == nil {
		return rideUnit{}
	}
	return env.height()
}

func newTx(env RideEnvironment) rideType {
	if env == nil {
		return rideUnit{}
	}
	tx := env.transaction()
	if tx == nil {
		return rideUnit{}
	}
	return tx
}

func newLastBlock(env RideEnvironment) rideType {
	if env == nil {
		return rideUnit{}
	}
	b := env.block()
	if b == nil {
		return rideUnit{}
	}
	return b
}

func newThis(env RideEnvironment) rideType {
	if env == nil {
		return rideUnit{}
	}
	this := env.this()
	if this == nil {
		return rideUnit{}
	}
	return this
}

func newInvocation(env RideEnvironment) rideType {
	if env == nil {
		return rideUnit{}
	}
	inv := env.invocation()
	if inv == nil {
		return rideUnit{}
	}
	return inv
}

func newUnit(RideEnvironment) rideType {
	return rideUnit{}
}

func newNil(RideEnvironment) rideType {
	return rideList(nil)
}
