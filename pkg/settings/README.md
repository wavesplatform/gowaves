# How to generate embedded settings files

Installation:

```bash
go get github.com/rakyll/statik
```

Execute following command:

```bash
statik -src pkg/settings/embedded/ -dest pkg/settings/ -p embedded -f
```