# chaincmp

Utility to compare blockchains on the node and reference nodes.

## How it works

`chaincmp` uses nodes API to get information about blocks in the blockchains. So, the utility can be used only if the APIs of the nodes are open.
In the beginning chaincmp detects the lowest height among the nodes. After that it starts to compare blocks IDs using binary search. 
If all blocks IDs are identical the node is on the same fork. If not, the utility finds the last common block and reports it.

## Usage and examples

```
usage: chaincmp [flags]
  -h, --help                Print usage information (this message) and quit
  -n, --node string         URL of the node
  -r, --references string   A list of space-separated URLs of reference nodes, for example "http://127.0.0.1:6869 https://nodes.wavesnodes.com" (default "https://nodes.wavesnodes.com")
      --silent              Produce no output except this help message; incompatible with "verbose"
      --verbose             Logs additional information; incompatible with "silent"
  -v, --version             Print version information and quit
```

In simple case you need to provide only the `-n` flag with the address of the node.

```bash
chaincmp -n http://127.0.0.1:6869
```

The default reference node (https://nodes.wavesnodes.com) will be used. If no `http` or `https` is given, the default protocol `http` will be used.
Other protocol are not supported and will lead to error.

For the scripting purposes the `--silent` flag is useful.

```bash
./chaincmp -n http://127.0.0.1:6869 --silent
```

In this case utility omits the output and produces only result code. If node is on the same fork it will exit with the 0 result code. 
If the node is on fork utility exits with 1 result code. In case of incorrect parameters or network errors it will exit with 2 result code.

To get more information about differences between chains use `--verbose` flag. In verbose mode `chaincmp` prints the IDs of compared blocks. 