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
	OpCall                     //0a - Call a function declared at given address. Two parameters: position of function declaration, number of arguments.
	OpGlobal                   //0b - Load global constant. One parameter: global constant ID.
	OpLoad                     //0c - Evaluates an expression that declared at address. One parameter: position of declaration.
	OpLoadLocal                //0d - Load an argument of function call on stack. One parameter: argument number.
	OpRef                      //0e - Put reference to expression/function on stack. One parameter: position of declaration.
)
