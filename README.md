Odu
====

Odu is a middleware program which helps with talking to external programs from Elixir or Erlang.

Port implementation in beam does not put any back-pressure on the external program which can lead to flooding of owner process mailbox with port command output. Odu tries to fix this by acting as a middleware and by making the output consumption as a demand driven. It also try to fill other gaps such as exiting external program properly.

Odu is heavily based on [goon](https://github.com/alco/goon) by [Alexei Sholik](https://github.com/alco). Changes are made on top of goon, odu changes are incompatible with goon.

## Usage

Put odu somewhere in your `PATH` (or into the directory that will become the current
working directory of your application).

## Building from source

```sh
$ go build
```

## License

This software is licensed under [the MIT license](LICENSE).
