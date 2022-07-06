package render

import (
	"image"

	"github.com/fogleman/gg"
)

// Marquee scrolls its child horizontally or vertically.
//
// The `scroll_direction` will be 'horizontal' and will scroll from right
// to left if left empty, if specified as 'vertical' the Marquee will
// scroll from bottom to top.
//
// In horizontal mode the height of the Marquee will be that of its child,
// but its `width` must be specified explicitly. In vertical mode the width
// will be that of its child but the `height` must be specified explicitly.
//
// If the child's width fits fully, it will not scroll.
//
// The `offset_start` and `offset_end` parameters control the position
// of the child in the beginning and the end of the animation.
//
// Alignment for a child that fits fully along the horizontal/vertical axis is controlled by passing
// one of the following `align` values:
// - `"start"`: place child at the left/top
// - `"end"`: place child at the right/bottom
// - `"center"`: place child at the center

// DOC(Child): Widget to potentially scroll
// DOC(Width): Width of the Marquee, required for horizontal
// DOC(Height): Height of the Marquee, required for vertical
// DOC(OffsetStart): Position of child at beginning of animation
// DOC(OffsetEnd): Position of child at end of animation
// DOC(ScrollDirection): Direction to scroll, 'vertical' or 'horizontal', default is horizontal
// DOC(Align): alignment when contents fit on screen, 'start', 'center' or 'end', default is start
//
// EXAMPLE BEGIN
// render.Marquee(
//      width=64,
//      child=render.Text("this won't fit in 64 pixels"),
//      offset_start=5,
//      offset_end=32,
// )
// EXAMPLE END
type Marquee struct {
	Widget
	Child           Widget `starlark:"child,required"`
	Width           int    `starlark:"width"`
	Height          int    `starlark:"height"`
	OffsetStart     int    `starlark:"offset_start"`
	OffsetEnd       int    `starlark:"offset_end"`
	ScrollDirection string `starlark:"scroll_direction"`
	Align           string `starlark:"align"`
}

func (m Marquee) FrameCount() int {
	var im image.Image
	var cw int
	var size int
	if m.isVertical() {
		im = m.Child.Paint(image.Rect(0, 0, DefaultFrameWidth, m.Height*10), 0)
		cw = im.Bounds().Dy()
		size = m.Height
	} else {
		im = m.Child.Paint(image.Rect(0, 0, m.Width*10, DefaultFrameHeight), 0)
		cw = im.Bounds().Dx()
		size = m.Width
	}

	if cw <= size {
		return 1
	}

	offstart := m.OffsetStart
	if offstart < -cw {
		offstart = -cw
	}

	offend := m.OffsetEnd
	if offend < -cw {
		offend = -cw
	}

	// If start and end offsets are identical, do not
	// repeat these identical frames after another.
	if offstart == offend {
		return cw + offstart + size - offend
	} else {
		return cw + offstart + size - offend + 1
	}
}

func (m Marquee) Paint(bounds image.Rectangle, frameIdx int) image.Image {
	var im image.Image
	var cw int
	var size int
	if m.isVertical() {
		// We'll only scroll frame 0 of the child. Scrolling an
		// animation would be madness.
		im = m.Child.Paint(image.Rect(0, 0, bounds.Dx(), m.Height*10), 0)
		cw = im.Bounds().Dy()
		size = m.Height
	} else {
		im = m.Child.Paint(image.Rect(0, 0, m.Width*10, bounds.Dy()), 0)
		cw = im.Bounds().Dx()
		size = m.Width
	}

	offstart := m.OffsetStart
	if offstart < -cw {
		offstart = -cw
	}

	offend := m.OffsetEnd
	if offend < -cw {
		offend = -cw
	}

	loopIdx := cw + offstart
	endIdx := cw + offstart + size - offend

	align := 0.0 //default is align="start"
	var offset int
	if cw <= size {
		// child fits entirely and we don't want to scroll it anyway
		offset = 0

		//modify alignment
		if m.Align == "center" {
			align = 0.5
			offset = size / 2
		} else if m.Align == "end" {
			align = 1.0
			offset = size
		}
	} else if frameIdx <= loopIdx {
		// first scroll child out of view
		offset = offstart - frameIdx
	} else if frameIdx <= endIdx {
		// then, scroll back into view
		offset = offend + (endIdx - frameIdx)
	} else {
		// if more than FrameCount frames are requested,
		// freeze marquee at final frame
		offset = offend
	}

	var dc *gg.Context
	if m.isVertical() {
		dc = gg.NewContext(im.Bounds().Dx(), m.Height)
		dc.DrawImageAnchored(im, 0, offset, 0, align)
	} else {
		dc = gg.NewContext(m.Width, im.Bounds().Dy())
		dc.DrawImageAnchored(im, offset, 0, align, 0)
	}

	return dc.Image()
}

func (m Marquee) isVertical() bool {
	return m.ScrollDirection == "vertical"
}
