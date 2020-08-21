package fride

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
	if env.transaction() == nil {
		return rideUnit{}
	}
	return env.transaction()
}

func newLastBlock(env RideEnvironment) rideType {
	if env == nil {
		return rideUnit{}
	}
	if env.block() == nil {
		return rideUnit{}
	}
	return env.block()
}

func newThis(env RideEnvironment) rideType {
	if env == nil {
		return rideUnit{}
	}
	if env.this() == nil {
		return rideUnit{}
	}
	return env.this()
}

func newUnit(RideEnvironment) rideType {
	return rideUnit{}
}

func newNil(RideEnvironment) rideType {
	return nil
}
