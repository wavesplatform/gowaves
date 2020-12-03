package ride

// Operation code is 1 byte.
// Parameter is 2 bytes length

const (
	OpHalt         byte = iota //00 - Halts program execution. No parameters.
	OpReturn                   //01 - Returns from declaration to stored position. No parameters.
	OpPush                     //02 - Put constant on stack. One parameter: constant ID.
	OpPop                      //03 - Removes value from stack. No parameters.
	OpTrue                     //04 - Put True value on stack. No parameters.
	OpFalse                    //05 - Put False value on stack. No parameters.
	OpJump                     //06 - Moves instruction pointer to new position. One parameter: new position.
	OpJumpIfFalse              //07 - Moves instruction pointer to new position if value on stack is False. One parameter: new position.
	OpProperty                 //08 - Puts value of object's property on stack. One parameter: constant ID that holds name of the property.
	OpExternalCall             //09 - Call a standard library function. Two parameters: function ID, number of arguments.
	OpCall                     //10 0xa - Call a function declared at given address. One parameter: position of function declaration.
	OpSetArg                   //11 0xb - FROM (global) -> TO (local): Set value into cell. Two parameters: constant id and cell id.
	OpCache                    //12 0xc - Put constant on stack. One parameter: constant ID.
	OpRef                      //14 0xd = ref id
	OpClearCache               //15 0xe = ref id

	// odd, will be removed.
	OpGlobal
	OpLoadLocal
	OpLoad
)
