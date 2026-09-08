# Ride Code Generation

## Tooling

Code-generation tool located at `pkg/ride/generate/main.go`. It uses following configuration files:
 * `pkg/ride/compiler/stdlib/funcs.json` - signatures of standard library functions
 * `pkg/ride/compiler/stdlib/ride_objects.json` - descriptions of Ride objects
 * `pkg/ride/compiler/stdlib/vars.json` - declarations of global variables

Run string for Ride code generation located at the top of `pkg/ride/compiler/compiler.go` file. 

## Describing Ride Objects

To add a new Ride object, you need to edit `ride_objects.json` file.
Each object is described by the following fields:
 * `name` - name of the object
 * `actions` - list of code-generation actions to perform on the object
   * `version` - library version when the object was added or modified
   * `deleted` - library version when the object was deleted (optional)
   * `fields` - list of fields in the object
     * `name` - name of the field
     * `type` - type of the field (see Ride types)
     * `order` - field position in the string representation of the object
     * `constructorOrder` - field position in the constructor arguments list

Constructor order is required to match the order of arguments in the Scala implementation of the object constructor.
String representation order is optional and only required for the early versions of Ride libraries (prior V6), 
see Waves PRs [#3625](https://github.com/wavesplatform/Waves/pull/3625) and [#3627](https://github.com/wavesplatform/Waves/pull/3627).
For object introduced after V6 the string representation order can be set the same as the constructor order or left 0.
