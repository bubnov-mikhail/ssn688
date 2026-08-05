package assets

import "embed"

//go:embed menu_bg.jpg
var MenuBG []byte

//go:embed bathy.bin
var BathyChart []byte

//go:embed library/*.jpg
var LibraryPhotos embed.FS
