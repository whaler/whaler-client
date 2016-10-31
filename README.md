# Whaler client

Client for [Whaler](https://github.com/whaler/whaler)

## Build

### Linux 64bit

```sh
$ docker run -it --rm -v $HOME:$HOME -w `pwd` -e GOOS=linux golang:1.6 ./build.sh
```

### Darwin/macOS 64bit

```sh
$ docker run -it --rm -v $HOME:$HOME -w `pwd` -e GOOS=darwin golang:1.6 ./build.sh
```

### Windows 64bit

```sh
$ docker run -it --rm -v $HOME:$HOME -w `pwd` -e GOOS=windows golang:1.6 ./build.sh
```

## License

This software is under the MIT license. See the complete license in:

```
LICENSE
```
