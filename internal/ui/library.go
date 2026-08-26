package ui

import (
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ssn688/sim/assets"
	"github.com/ssn688/sim/internal/acoustics"
	"github.com/ssn688/sim/internal/i18n"
	"github.com/ssn688/sim/internal/layout"
	"github.com/ssn688/sim/internal/render"
	"github.com/ssn688/sim/internal/world"
)

const (
	libPanelX = 20
	libPanelY = 50
	libPanelW = 1260
	libPanelH = 700

	libLeftX = 40
	libLeftW = 400
	libRowH  = 20

	libCatY = 118
	libCatH = 278

	libConY = 420
	libConH = 278

	libRightX = 460
	libRightY = 118
	libRightW = 800
	libRightH = 580

	libPhotoH = 150
	libConVis = 12
)

var (
	libraryRowsOnce  sync.Once
	libraryRowsCache []libraryTableRow

	libraryPhotoOnce sync.Map // id -> *ebiten.Image or error sentinel
)

func libraryRows() []libraryTableRow {
	libraryRowsOnce.Do(func() {
		libraryRowsCache = libraryTableRows()
	})
	return libraryRowsCache
}

func (a *App) ensureLibrarySelection() {
	if a.librarySelectedID != "" && libraryEntryByID(a.librarySelectedID) != nil {
		return
	}
	for _, r := range libraryRows() {
		if !r.Header && r.EntryID != "" {
			a.librarySelectedID = r.EntryID
			return
		}
	}
}

func (a *App) selectLibraryEntry(id string) {
	if libraryEntryByID(id) == nil {
		return
	}
	if a.librarySelectedID != id {
		a.libraryDetailScroll = 0
	}
	a.librarySelectedID = id
}

func (a *App) libraryPhoto(id string) *ebiten.Image {
	if v, ok := libraryPhotoOnce.Load(id); ok {
		if img, ok := v.(*ebiten.Image); ok {
			return img
		}
		return nil
	}
	e := libraryEntryByID(id)
	if e == nil || e.ImageFile == "" {
		libraryPhotoOnce.Store(id, false)
		return nil
	}
	raw, err := assets.LibraryPhotos.ReadFile("library/" + e.ImageFile)
	if err != nil {
		libraryPhotoOnce.Store(id, false)
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		libraryPhotoOnce.Store(id, false)
		return nil
	}
	eb := ebiten.NewImageFromImage(img)
	libraryPhotoOnce.Store(id, eb)
	return eb
}

func wrapLibraryText(s string, maxChars int) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if maxChars < 12 {
		maxChars = 12
	}
	words := strings.Fields(s)
	var lines []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len() == 0 {
			cur.WriteString(w)
			continue
		}
		if cur.Len()+1+len(w) > maxChars {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(w)
			continue
		}
		cur.WriteByte(' ')
		cur.WriteString(w)
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

func (a *App) libraryDetailLines(e *libraryEntry) []detailLine {
	if e == nil {
		return nil
	}
	maxChars := libRightW/7 - 2
	var out []detailLine
	addSection := func(title string, clr color.Color) {
		out = append(out, detailLine{Text: title, Color: clr, Section: true})
	}
	addBody := func(lines []i18n.TranslatedText, clr color.Color) {
		for _, para := range lines {
			for _, ln := range wrapLibraryText(a.L(para), maxChars) {
				out = append(out, detailLine{Text: ln, Color: clr})
			}
			out = append(out, detailLine{Text: "", Color: clr})
		}
	}
	addBullets := func(items []i18n.TranslatedText, clr color.Color) {
		for _, it := range items {
			wrapped := wrapLibraryText("• "+a.L(it), maxChars)
			for i, ln := range wrapped {
				if i > 0 {
					ln = "  " + ln
				}
				out = append(out, detailLine{Text: ln, Color: clr})
			}
		}
		out = append(out, detailLine{Text: "", Color: clr})
	}

	addSection(a.L(i18n.UIOverview), render.ColorAmber)
	addBody(e.Summary, render.ColorPlateLabel)
	addSection(a.L(i18n.UICharacteristics), render.ColorAmber)
	addBullets(e.Specs, render.ColorPlateLabel)
	addSection(a.L(i18n.UIOffensiveWeapons), render.ColorAmber)
	addBullets(e.Offense, render.ColorPlateLabel)
	addSection(a.L(i18n.UIDefensiveSystems), render.ColorAmber)
	addBullets(e.Defense, render.ColorPlateLabel)
	addSection(a.L(i18n.UINeutralization), render.ColorWarn)
	addBullets(e.Neutralize, render.ColorPlateLabel)
	addSection(a.L(i18n.UIEvasion), render.ColorWarn)
	addBullets(e.Evade, render.ColorPlateLabel)
	if a.L(e.Credit) != "" {
		addSection(a.L(i18n.UIImageCredit), render.ColorPlateLabel)
		addBody([]i18n.TranslatedText{e.Credit}, render.ColorDim)
	}
	return out
}

type detailLine struct {
	Text    string
	Color   color.Color
	Section bool
}

func (a *App) updateLibraryInput() {
	if a.Engine == nil {
		return
	}
	a.validateSelectedContact(&a.Engine.Sonar)
	a.ensureLibrarySelection()

	mx, my := ebiten.CursorPosition()
	rows := libraryRows()
	catVis := libCatH / libRowH
	a.libraryCatalogScroll = clampContactTableScroll(a.libraryCatalogScroll, len(rows), catVis)
	scrollContactTableWheel(mx, my, libLeftX, libCatY, libLeftW, libCatH, len(rows), catVis, &a.libraryCatalogScroll)

	sonar := &a.Engine.Sonar
	a.contactTableScroll.library = clampContactTableScroll(a.contactTableScroll.library, len(sonar.Contacts), libConVis)
	scrollContactTableWheel(mx, my, libLeftX, libConY, libLeftW, libConH, len(sonar.Contacts), libConVis, &a.contactTableScroll.library)

	// Detail pane scroll when hovering right column.
	if e := libraryEntryByID(a.librarySelectedID); e != nil {
		lines := a.libraryDetailLines(e)
		textTop := libRightY + 28 + libPhotoH + 28
		textH := libRightY + libRightH - textTop
		vis := max(1, textH/16)
		if inRect(mx, my, libRightX, textTop, libRightW, textH) {
			_, wheelY := ebiten.Wheel()
			if math.Abs(wheelY) >= 0.01 {
				step := int(math.Ceil(math.Abs(wheelY)))
				if wheelY < 0 {
					a.libraryDetailScroll += step
				} else {
					a.libraryDetailScroll -= step
				}
			}
		}
		a.libraryDetailScroll = clampContactTableScroll(a.libraryDetailScroll, len(lines), vis)
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	// Catalog row click.
	start, end := contactTableWindow(len(rows), a.libraryCatalogScroll, catVis)
	y := libCatY
	for i := start; i < end; i++ {
		r := rows[i]
		if !r.Header && mx >= libLeftX && mx < libLeftX+libLeftW && my >= y && my < y+libRowH {
			a.selectLibraryEntry(r.EntryID)
			a.uiPressedID = "lib_cat"
			a.uiPressedAt = time.Now()
			return
		}
		y += libRowH
	}

	// Contact row click — classified contacts drive catalog selection.
	cStart, cEnd := contactTableWindow(len(sonar.Contacts), a.contactTableScroll.library, libConVis)
	y = libConY + libRowH
	for i := cStart; i < cEnd; i++ {
		if mx >= libLeftX && mx < libLeftX+libLeftW && my >= y && my < y+libRowH {
			c := &sonar.Contacts[i]
			a.selectContact(sonar, c)
			if id := classifiedLibraryID(c); id != "" {
				a.selectLibraryEntry(id)
			}
			return
		}
		y += libRowH
	}
}

func classifiedLibraryID(c *acoustics.Contact) string {
	if c == nil {
		return ""
	}
	if c.ConfirmedID != "" {
		if libraryEntryByID(c.ConfirmedID) != nil {
			return c.ConfirmedID
		}
	}
	if c.ConfirmedClass == "" {
		return ""
	}
	for _, e := range libraryCatalog {
		if p, ok := world.ProfileByID(e.ID); ok {
			if p.MatchesLabel(c.ConfirmedClass) {
				return e.ID
			}
		}
		for _, lang := range i18n.SupportedLangs {
			if e.Title.GetText(lang) == c.ConfirmedClass {
				return e.ID
			}
		}
	}
	return ""
}

func (a *App) drawLibrary(screen *ebiten.Image) {
	sonar := &a.Engine.Sonar
	a.ensureLibrarySelection()
	rows := libraryRows()
	catVis := libCatH / libRowH

	render.DrawConsolePanel(screen, libPanelX, libPanelY, libPanelW, libPanelH)
	render.DrawScreenTitle(screen, a.L(i18n.UIThreatLibrary), layout.PassiveTitleLabelX, layout.PassiveTitleLabelY+20)

	// —— Catalog table ——
	render.DrawText(screen, a.L(i18n.UIPlatformTypes), libLeftX, libCatY-14, render.ColorPlateLabel, true)
	render.FillRect(screen, libLeftX, libCatY, libLeftW, libCatH, render.ColorPanelInset)
	a.libraryCatalogScroll = clampContactTableScroll(a.libraryCatalogScroll, len(rows), catVis)
	start, end := contactTableWindow(len(rows), a.libraryCatalogScroll, catVis)
	mx, my := ebiten.CursorPosition()
	y := libCatY
	for i := start; i < end; i++ {
		r := rows[i]
		if r.Header {
			render.FillRect(screen, libLeftX+1, y, libLeftW-2, libRowH, color.RGBA{36, 40, 48, 255})
			render.DrawText(screen, "— "+a.L(r.Label)+" —", libLeftX+8, y+14, render.ColorAmber, true)
		} else {
			selected := r.EntryID == a.librarySelectedID
			hover := mx >= libLeftX && mx < libLeftX+libLeftW && my >= y && my < y+libRowH
			if selected {
				render.FillRect(screen, libLeftX+2, y, libLeftW-4, libRowH, color.RGBA{80, 60, 0, 180})
			} else if hover {
				render.FillRect(screen, libLeftX+2, y, libLeftW-4, libRowH, render.ColorPanelMid)
			}
			clr := render.ColorPlateLabel
			if selected {
				clr = render.ColorAmber
			}
			label := a.L(r.Label)
			if len(label) > 48 {
				label = label[:45] + "..."
			}
			render.DrawText(screen, label, libLeftX+10, y+14, clr, true)
		}
		y += libRowH
	}
	drawContactTableScrollbar(screen, libLeftX+libLeftW+4, libCatY, libCatH, len(rows), catVis, a.libraryCatalogScroll)

	// —— Contacts table ——
	render.DrawText(screen, a.L(i18n.UIContacts), libLeftX, libConY-14, render.ColorPlateLabel, true)
	render.FillRect(screen, libLeftX, libConY, libLeftW, libConH, render.ColorPanelInset)
	render.DrawText(screen, a.L(i18n.UIColID), libLeftX+8, libConY+14, render.ColorPhosphorDim, true)
	render.DrawText(screen, a.L(i18n.UIColBRG), libLeftX+56, libConY+14, render.ColorPhosphorDim, true)
	render.DrawText(screen, a.L(i18n.UIColClass), libLeftX+110, libConY+14, render.ColorPhosphorDim, true)
	a.contactTableScroll.library = clampContactTableScroll(a.contactTableScroll.library, len(sonar.Contacts), libConVis)
	cStart, cEnd := contactTableWindow(len(sonar.Contacts), a.contactTableScroll.library, libConVis)
	y = libConY + libRowH
	for i := cStart; i < cEnd; i++ {
		c := &sonar.Contacts[i]
		selected := c.SourceEntityID == a.selectedContactID
		hover := mx >= libLeftX && mx < libLeftX+libLeftW && my >= y && my < y+libRowH
		if selected {
			render.FillRect(screen, libLeftX+2, y, libLeftW-4, libRowH, color.RGBA{80, 60, 0, 180})
		} else if hover {
			render.FillRect(screen, libLeftX+2, y, libLeftW-4, libRowH, render.ColorPanelMid)
		}
		clr := render.ColorPlateLabel
		class := contactClassLabel(c)
		if c.ConfirmedClass != "" || c.ConfirmedID != "" {
			clr = render.ColorAmber
		}
		render.DrawText(screen, c.ID, libLeftX+8, y+14, clr, true)
		render.DrawText(screen, contactBearingLabel(c), libLeftX+56, y+14, clr, true)
		if len(class) > 36 {
			class = class[:33] + "..."
		}
		render.DrawText(screen, class, libLeftX+110, y+14, clr, true)
		y += libRowH
	}
	drawContactTableScrollbar(screen, libLeftX+libLeftW+4, libConY+libRowH, libConH-libRowH, len(sonar.Contacts), libConVis, a.contactTableScroll.library)

	// —— Detail pane ——
	a.drawLibraryDetail(screen)

	render.DrawText(screen, a.L(i18n.UILibFooter), 40, 720, render.ColorDim, true)
}

func (a *App) drawLibraryDetail(screen *ebiten.Image) {
	e := libraryEntryByID(a.librarySelectedID)
	render.FillRect(screen, libRightX, libRightY, libRightW, libRightH, render.ColorPanelInset)
	if e == nil {
		render.DrawText(screen, a.L(i18n.UISelectPlatform), libRightX+16, libRightY+24, render.ColorDim, true)
		return
	}

	render.DrawText(screen, a.L(e.Title), libRightX+12, libRightY+20, render.ColorAmber, false)

	photoY := libRightY + 32
	photo := a.libraryPhoto(e.ID)
	render.FillRect(screen, libRightX+12, photoY, libRightW-24, libPhotoH, color.RGBA{0, 0, 0, 255})
	if photo != nil {
		pw, ph := photo.Bounds().Dx(), photo.Bounds().Dy()
		if pw > 0 && ph > 0 {
			boxW := float64(libRightW - 24)
			boxH := float64(libPhotoH)
			scale := math.Min(boxW/float64(pw), boxH/float64(ph))
			op := &ebiten.DrawImageOptions{}
			op.Filter = ebiten.FilterLinear
			op.GeoM.Scale(scale, scale)
			dw := float64(pw) * scale
			dh := float64(ph) * scale
			op.GeoM.Translate(float64(libRightX+12)+(boxW-dw)/2, float64(photoY)+(boxH-dh)/2)
			screen.DrawImage(photo, op)
		}
	} else {
		render.DrawText(screen, a.L(i18n.UINoImage), libRightX+libRightW/2-40, photoY+libPhotoH/2, render.ColorDim, true)
	}

	textTop := photoY + libPhotoH + 16
	textH := libRightY + libRightH - textTop - 8
	lines := a.libraryDetailLines(e)
	vis := max(1, textH/16)
	a.libraryDetailScroll = clampContactTableScroll(a.libraryDetailScroll, len(lines), vis)
	start, end := contactTableWindow(len(lines), a.libraryDetailScroll, vis)
	y := textTop + 12
	for i := start; i < end; i++ {
		ln := lines[i]
		clr := ln.Color
		if ln.Section {
			clr = render.ColorAmber
		}
		render.DrawText(screen, ln.Text, libRightX+14, y, clr, true)
		y += 16
	}
	drawContactTableScrollbar(screen, libRightX+libRightW-6, textTop, textH, len(lines), vis, a.libraryDetailScroll)
}
