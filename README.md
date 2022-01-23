# Whaler client

Client for [Whaler](https://github.com/whaler/whaler)

## File Sharing

By default `client` provide access for `HOME` directory. If you need access for different directories, 
create `~/.whaler/client.json` file with this content:

```json
{
    "file-sharing": [
        "/some/other/path"
    ]
}
```

## Build

```sh
docker run -it --rm -v $HOME:$HOME -w `pwd` golang:1.17 ./scripts/make.sh
```

## License

This software is under the MIT license. See the complete license in:

```
LICENSE
```
