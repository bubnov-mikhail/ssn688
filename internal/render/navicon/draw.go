package navicon

import "math"

func drawPassive(c *Canvas) {
	cx, cy := 64.0, 72.0
	for _, deg := range []float64{-55, -28, 0, 28, 55} {
		rad := (deg - 90) * math.Pi / 180
		c.line(cx, cy, cx+math.Cos(rad)*44, cy+math.Sin(rad)*44, 2.5)
	}
	c.line(28, 92, 100, 92, 2.5)
	c.disk(cx, cy, 5)
}

func drawActive(c *Canvas) {
	cx, cy := 64.0, 64.0
	for _, r := range []float64{20, 36, 52} {
		c.circle(cx, cy, r, 2.5)
	}
	c.fillRect(61, 61, 67, 67)
}

func drawSpectrum(c *Canvas) {
	bars := []int{28, 44, 36, 56, 40, 48, 32}
	x0, base := 28, 96
	for i, h := range bars {
		x := x0 + i*12
		c.fillRect(x, base-h, x+8, base)
	}
	c.line(20, 100, 108, 100, 2.5)
}

func drawLibrary(c *Canvas) {
	c.rect(32, 28, 96, 100, 2.5)
	c.line(64, 28, 64, 100, 2.5)
	for _, y := range []float64{44, 60, 76} {
		c.line(40, y, 56, y, 1.8)
		c.line(72, y, 88, y, 1.8)
	}
}

func drawWeapons(c *Canvas) {
	c.drawTorpedo(64, 46, 0, 54, 9)
	c.drawTorpedo(64, 64, 0, 58, 10)
	c.drawTorpedo(64, 82, 0, 54, 9)
}

func drawManeuver(c *Canvas) {
	cx, cy := 64.0, 64.0
	c.circle(cx, cy, 44, 2.5)
	for deg := 0; deg < 360; deg += 45 {
		rad := float64(deg-90) * math.Pi / 180
		c.line(cx, cy, cx+math.Cos(rad)*36, cy+math.Sin(rad)*36, 2.5)
	}
	c.disk(cx, cy, 10)
}

func drawMast(c *Canvas) {
	c.rect(40, 90, 88, 106, 2.5)
	c.rect(50, 52, 78, 90, 2.5)
	c.circle(57, 46, 7, 2.2)
	c.circle(71, 46, 7, 2.2)
	c.line(57, 46, 71, 46, 2)
	c.line(50, 66, 22, 66, 2.5)
	c.line(18, 66, 26, 66, 2.5)
	c.disk(22, 66, 4.5)
	c.line(78, 66, 106, 66, 2.5)
	c.line(102, 66, 110, 66, 2.5)
	c.disk(106, 66, 4.5)
	c.circle(64, 78, 9, 2)
	c.line(64, 78, 64, 69, 1.8)
	c.line(64, 78, 71, 78, 1.8)
}

func drawDamage(c *Canvas) {
	const sw = 2.3
	const lineW = 1.8

	cx := float64(DesignSize) * 2 / 3
	top, bot := 28.0, 108.0
	halfW := 22.0
	bodyL := cx - halfW
	bodyR := cx + halfW
	r := halfW

	yTop := top + r
	yBot := bot - r
	c.solidArc(cx, yTop, r, math.Pi, 2*math.Pi, sw)
	c.solidLine(bodyL, yTop, bodyL, yBot, sw)
	c.solidLine(bodyR, yTop, bodyR, yBot, sw)
	c.solidArc(cx, yBot, r, 0, math.Pi, sw)

	midY := (top + bot) / 2
	inset := 5.0
	c.solidLine(bodyL+inset, midY-5, bodyR-inset, midY-5, lineW)
	c.solidLine(bodyL+inset, midY+5, bodyR-inset, midY+5, lineW)

	neckHalfW := 7.0
	capHalfW := 15.0
	neckTop := top - 14.0
	capTop := neckTop - 8.0
	c.solidLine(cx-neckHalfW, neckTop, cx-neckHalfW, top, sw)
	c.solidLine(cx+neckHalfW, neckTop, cx+neckHalfW, top, sw)
	c.solidRect(cx-capHalfW, capTop, cx+capHalfW, neckTop, sw)

	hornAttachX := cx - neckHalfW
	hornMid := (neckTop+top)/2 + 3
	hornMouthX := 20.0
	c.solidPolygon([][2]float64{
		{hornAttachX, hornMid - 2.5},
		{hornAttachX, hornMid + 2.5},
		{hornMouthX, hornMid + 10},
		{hornMouthX, hornMid - 10},
	}, sw)

	const flameSide = 58.0
	const flameSW = 2.6
	flameH := flameSide * math.Sqrt(3) / 2
	flameBaseY := 116.0
	flameApexY := flameBaseY - flameH
	flameCx := bodyL - 0.2*flameSide
	flamePts := [][2]float64{
		{flameCx, flameApexY},
		{flameCx - flameSide/2, flameBaseY},
		{flameCx + flameSide/2, flameBaseY},
	}
	c.clearInsidePolygon(flamePts)
	c.solidPolygon(flamePts, flameSW)

	flameCy := (flameApexY + 2*flameBaseY) / 3
	markCy := flameCy - 2
	c.solidLine(flameCx, markCy-8, flameCx, markCy+1, 2.2)
	c.solidDisk(flameCx, markCy+7.5, 2.5)
}

func drawTactical(c *Canvas) {
	c.rect(24, 24, 104, 104, 2.5)
	c.line(64, 32, 64, 96, 1.5)
	c.line(32, 64, 96, 64, 1.5)
	c.polygon([][2]float64{{64, 56}, {56, 72}, {72, 72}}, 2)
	c.fillRect(56, 72, 72, 74)
	c.disk(80, 44, 4)
	c.disk(48, 76, 4)
}
