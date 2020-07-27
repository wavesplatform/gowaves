package fride

const (
	OpPush         byte = iota //00 - 3
	OpPop                      //01 - 1
	OpTrue                     //02 - 1
	OpFalse                    //03 - 1
	OpJump                     //04 - 3
	OpJumpIfFalse              //05 - 3
	OpProperty                 //06 - 3
	OpCall                     //07 - 2
	OpExternalCall             //08 - 3 Params: functionID - byte, argsCount - byte
	OpStore                    //09 - 3
	OpLoad                     //0a - 3
	OpReturn                   //0b - 1
	OpRecord                   //0c - 5
)
