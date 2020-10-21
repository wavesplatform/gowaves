package ride


type TemporaryStateIface interface {
	//NewStateFromOld(t TemporaryStateT) TemporaryStateT
	GetCurrentEnv()
	Apply()
}

type  TemporaryStateT struct {
	tempEnv RideEnvironment
	methods TemporaryStateIface
}


//func (state TemporaryStateT) NewStateFromOld() TemporaryStateT{
//	return TemporaryStateT{env: state.env, methods: state.methods}
//}

func (state TemporaryStateT) GetCurrentEnv() RideEnvironment {
	return state.tempEnv
}

func (state TemporaryStateT) Apply(env *RideEnvironment) {
	*env = state.tempEnv
}


func InitTemporaryState(newMethods TemporaryStateIface, newEnv RideEnvironment) TemporaryStateT{
	return TemporaryStateT{tempEnv: newEnv, methods: newMethods}
}





