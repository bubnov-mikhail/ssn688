package assets

import "embed"

//go:embed menu_bg.jpg
var MenuBG []byte

//go:embed app_icon/icon.png
var AppIconPNG []byte

//go:embed library/*.jpg
var LibraryPhotos embed.FS

// PeriShipSprites are grayscale IR silhouettes rendered from Blender
// (assets/peri_ships/{class}_{aspect}.png).
//
//go:embed peri_ships/*.png
var PeriShipSprites embed.FS
