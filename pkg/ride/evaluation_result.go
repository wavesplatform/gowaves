package ride

type EvaluationResult struct {
}

func (a EvaluationResult) IsInterrupted() bool {
	panic("unimplemented!")
}

func (a EvaluationResult) Interrupted() *Executable {
	panic("unimplemented!")
}

func (a EvaluationResult) IsResult() bool {
	panic("unimplemented!")
}

func (a EvaluationResult) Result() ScriptResult {
	panic("unimplemented!")
}

func (a EvaluationResult) IsError() bool {
	panic("unimplemented!")
}

func (a EvaluationResult) Error() error {
	panic("unimplemented!")
}
