package fride

func newTx(env rideEnvironment) rideType {
	if env == nil {
		return rideUnit{}
	}
	if env.transaction() == nil {
		return rideUnit{}
	}
	return env.transaction()
}

func newLastBlock(env rideEnvironment) rideType {
	if env == nil {
		return rideUnit{}
	}
	if env.block() == nil {
		return rideUnit{}
	}
	return env.block()
}

func newThis(env rideEnvironment) rideType {
	if env == nil {
		return rideUnit{}
	}
	if env.this() == nil {
		return rideUnit{}
	}
	return env.this()
}

func newUnit(rideEnvironment) rideType {
	return rideUnit{}
}

func newNil(rideEnvironment) rideType {
	return nil
}
