package fride

const (
	OpPush         byte = iota //00 - 3 Put constant on stack, parameter: constant ID (2 bytes)
	OpPop                      //01 - 1 Removes value from stack
	OpTrue                     //02 - 1 Put True value on stack
	OpFalse                    //03 - 1 Put False value on stack
	OpJump                     //04 - 3 Moves instruction pointer to new position, parameter: new position (2 bytes)
	OpJumpIfFalse              //05 - 3 Moves instruction pointer to new position if value on stack is False, parameter: new position (2 bytes)
	OpProperty                 //06 - 3 Puts value of object's property on stack, parameter: constant ID that holds name of the property (2 bytes)
	OpCall                     //07 - 3 Call a function declared at given address, parameter: position of function declaration (2 bytes)
	OpExternalCall             //08 - 3 Call a built-in function, parameters: function ID (1 byte), number of arguments (1 byte)
	OpLoad                     //09 - 3 Load a value declared at address, parameter: position of declaration (2 bytes)
	OpLoadLocal                //0a - 3 Load an argument of function call on stack at given position, parameter: position on stack (2 bytes)
	OpReturn                   //0b - 1 Returns from declaration to stored position
	OpHalt                     //0c - 1 Halts execution immediately
	OpGlobal                   //0d - 2 Load global constant, parameter: global constant ID (1 byte)
)
