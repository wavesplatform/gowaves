package fride

const (
	OpPush         byte = iota //00 - 3
	OpPop                      //01 - 1
	OpTrue                     //02 - 1
	OpFalse                    //03 - 1
	OpJump                     //04 - 3
	OpJumpIfFalse              //05 - 3
	OpProperty                 //06 - 3
	OpCall                     //07 - 3
	OpExternalCall             //08 - 3 Params: functionID (1 byte), argsCount (1 byte)
	OpLoad                     //09 - 3 Params: addr (2 bytes)
	OpLoadLocal                //0a - 3 Params: position (2 bytes)
	OpReturn                   //0b - 1
)
