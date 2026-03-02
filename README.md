# rsbrTools
some tools for run sackboy run\
tested only for android v1.0.4 files, it probably won't work properly on other versions

## basic usage
run `rsbrTools --help` for full usage info

### unpack a pak file
`rsbrTools unpack ./pak.obb ./out`

### unpack a pak file, converting all textures
`rsbrTools unpack ./pak.obb ./out --convert`

### pack a directory to a pak file
`rsbrTools pack ./directory ./pak.obb`

### convert a .atc texture to .png and back
`rsbrTools atcdecode ./texture.atc ./decoded.png`\
`rsbrTools atcencode ./texture.png ./encoded.atc`

## thanks :)
[Ekey](https://github.com/Ekey) for making [RSBR.PAK.Tool](https://github.com/Ekey/RSBR.PAK.Tool)\
ETC1 decoder code is based off of [texture2ddecoder](https://github.com/K0lb3/texture2ddecoder)