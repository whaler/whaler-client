# Whaler client

Client for [Whaler](https://github.com/whaler/whaler)

## File Sharing

By default `client` provide access for `HOME` directory. If you need access for different directories, create `~/.whaler/client.json` file with this content:

```json
{
    "file-sharing": [
        "/some/other/path"
    ]
}
```

## Build

### Linux 64bit

```sh
$ docker run -it --rm -v $HOME:$HOME -w `pwd` -e GOOS=linux golang:1.9 ./build.sh
```

### Darwin/macOS 64bit

```sh
$ docker run -it --rm -v $HOME:$HOME -w `pwd` -e GOOS=darwin golang:1.9 ./build.sh
```

### Windows 64bit

```sh
$ docker run -it --rm -v $HOME:$HOME -w `pwd` -e GOOS=windows golang:1.9 ./build.sh
```

## License

This software is under the MIT license. See the complete license in:

```
LICENSE
```
