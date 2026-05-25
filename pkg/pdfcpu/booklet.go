/*
	Copyright 2021 The pdfcpu Authors.

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package pdfcpu

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

var errInvalidBookletAdvanced = errors.New("pdfcpu booklet advanced cannot have binding along the top (portrait short-edge, landscape long-edge). use plain booklet instead.")

var NUpValuesForBooklets = []int{2, 4, 6, 8, 10}

// DefaultBookletConfig returns the default configuration for a booklet
func DefaultBookletConfig() *model.NUp {
	nup := model.DefaultNUpConfig()
	nup.Margin = 0
	nup.Border = false
	nup.BookletGuides = false
	nup.MultiFolio = false
	nup.FolioSize = 8
	nup.BookletType = model.Booklet
	nup.BookletBinding = model.LongEdge
	nup.Enforce = true
	return nup
}

// PDFBookletConfig returns an NUp configuration for booklet-ing PDF files.
func PDFBookletConfig(val int, desc string, conf *model.Configuration) (*model.NUp, error) {
	nup := DefaultBookletConfig()
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	nup.InpUnit = conf.Unit
	if desc != "" {
		if err := ParseNUpDetails(desc, nup); err != nil {
			return nil, err
		}
	}
	if !types.IntMemberOf(val, NUpValuesForBooklets) {
		ss := make([]string, len(NUpValuesForBooklets))
		for i, v := range NUpValuesForBooklets {
			ss[i] = strconv.Itoa(v)
		}
		return nil, errors.Errorf("pdfcpu: n must be one of %s", strings.Join(ss, ", "))
	}
	if err := ParseNUpValue(val, nup); err != nil {
		return nil, err
	}
	// 6up special cases
	if nup.IsBooklet() && val == 6 && nup.IsTopFoldBinding() {
		// You can't top fold a 6up with 3 rows.
		return nup, fmt.Errorf("pdfcpu booklet: n=6 must have binding on side (portrait long-edge or landscape short-edge)")
	}
	// bookletadvanced
	if nup.BookletType == model.BookletAdvanced && val == 4 && nup.IsTopFoldBinding() {
		return nup, errInvalidBookletAdvanced
	}
	return nup, nil
}

// ImageBookletConfig returns an NUp configuration for booklet-ing image files.
func ImageBookletConfig(val int, desc string, conf *model.Configuration) (*model.NUp, error) {
	nup, err := PDFBookletConfig(val, desc, conf)
	if err != nil {
		return nil, err
	}
	nup.ImgInputFile = true
	return nup, nil
}

// input: positionNumber in the output grid
// output: original pdf page number and rotation, for this grid position
type pageNumberFunction func(positionNumber int, pageCount int, nup *model.NUp) (pageIndex int, rotated bool)

func nup2OutputPageNr(pos, pageCount int, nup *model.NUp) (int, bool) {
	// (pos+1, pageNr) = [(1,n), (2,1), (3, n-1), (4, 2), (5, n-2), (6, 3), ...] -- for portrait
	// for landscape, flip the above left-right, ie [(1, 1), (2, n), ...]
	isPortrait := nup.PageDim.Portrait()
	countDown := pageCount - 1 - pos/2
	countUp := pos / 2
	var p int
	if pos%2 == 0 {
		if isPortrait {
			p = countDown // top
		} else {
			p = countUp // left
		}
	} else {
		if isPortrait {
			p = countUp // bottom
		} else {
			p = countDown // right
		}
	}

	var rotate bool
	if pos%4 < 2 {
		// Rotate pages on the odd output sheets (the back sides) by 180 degrees
		// rotated pages are oriented with the bottoms on the right
		rotate = true
	}
	return p, rotate
}

func get4upPos(pos int, isLandscape bool) (out int) {
	if isLandscape {
		switch pos % 4 {
		// landscape short-edge binding page ordering is rotated 90 degrees anti-clockwise from the portrait ordering on the back sides of the pages to make duplexing work
		// from portrait to lanscape map {0 => 3, 1 => 2, 2 => 1, 3 => 0}
		case 0:
			return 3
		case 1:
			return 2
		case 2:
			return 1
		case 3:
			return 0
		}
	}
	return pos % 4
}

func nup4OutputPageNr(inputPageNr int, pageCount int, nup *model.NUp) (int, bool) {
	switch nup.BookletType {
	case model.Booklet:
		// simple booklets are collated by collecting the top of the sheet, then the bottom, then the top of the next sheet, and so on.
		// this is conceptually easier for collation without specialized tools.
		if nup.IsTopFoldBinding() {
			return nup4BasicTopFoldOutputPageNr(inputPageNr, pageCount, nup)
		} else {
			return nup4BasicSideFoldOutputPageNr(inputPageNr, pageCount, nup)
		}
	case model.BookletAdvanced:
		// advanced booklets have a different collation pattern: collect the top of each sheet and then the bottom of each sheet.
		// this allows printers to fold the sheets twice and then cut along one of the folds.
		return nup4AdvancedSideFoldOutputPageNr(inputPageNr, pageCount, nup)
	}
	return 0, false
}

func nup4BasicSideFoldOutputPageNr(positionNumber int, inputPageCount int, nup *model.NUp) (int, bool) {
	var p int
	bookletSheetSideNumber := positionNumber / 4
	bookletPageNumber := positionNumber / 8
	if bookletSheetSideNumber%2 == 0 {
		// front side
		n := bookletPageNumber * 4
		switch positionNumber % 4 {
		case 0:
			p = inputPageCount - n
		case 1:
			p = 1 + n
		case 2:
			p = 3 + n
		case 3:
			p = inputPageCount - 2 - n
		}
	} else {
		// back side
		n := bookletPageNumber * 4
		switch get4upPos(positionNumber, nup.PageDim.Landscape()) {
		case 0:
			p = 2 + n
		case 1:
			p = inputPageCount - 1 - n
		case 2:
			p = inputPageCount - 3 - n
		case 3:
			p = 4 + n
		}
	}
	// Rotate bottom row of each output sheet by 180 degrees.
	var rotate bool
	if positionNumber%4 >= 2 {
		rotate = true
	}
	return p - 1, rotate // p is one-indexed and we want zero-indexed
}

func nup4BasicTopFoldOutputPageNr(positionNumber int, inputPageCount int, nup *model.NUp) (int, bool) {
	var p int
	bookletSheetSideNumber := positionNumber / 4
	bookletSheetNumber := positionNumber / 8
	if bookletSheetSideNumber%2 == 0 {
		// front side
		switch positionNumber % 4 {
		case 0:
			p = inputPageCount - 4*bookletSheetNumber
		case 1:
			p = 3 + 4*bookletSheetNumber
		case 2:
			p = 1 + 4*bookletSheetNumber
		case 3:
			p = inputPageCount - 2 - 4*bookletSheetNumber
		}
	} else {
		// back side
		switch get4upPos(positionNumber, nup.PageDim.Landscape()) {
		case 0:
			p = 4 + 4*bookletSheetNumber
		case 1:
			p = inputPageCount - 1 - 4*bookletSheetNumber
		case 2:
			p = inputPageCount - 3 - 4*bookletSheetNumber
		case 3:
			p = 2 + 4*bookletSheetNumber
		}
	}
	// Rotate right side of output page by 180 degrees.
	var rotate bool
	if positionNumber%2 == 1 {
		rotate = true
	}
	return p - 1, rotate // p is one-indexed and we want zero-indexed
}

func nup4AdvancedSideFoldOutputPageNr(inputPageNr int, inputPageCount int, nup *model.NUp) (int, bool) {
	// (output page, input page) = [(1,n), (2,1), (3, n/2+1), (4, n/2-0), (5, 2), (6, n-1), (7, n/2-1), (8, n/2+2) ...]
	bookletPageNumber := inputPageNr / 4
	var p int
	if bookletPageNumber%2 == 0 {
		// front side
		switch inputPageNr % 4 {
		case 0:
			p = inputPageCount - 1 - bookletPageNumber
		case 1:
			p = bookletPageNumber
		case 2:
			p = inputPageCount/2 + bookletPageNumber
		case 3:
			p = inputPageCount/2 - 1 - bookletPageNumber
		}
	} else {
		// back side (portrait)
		switch get4upPos(inputPageNr, nup.PageDim.Landscape()) {
		case 0:
			p = bookletPageNumber
		case 1:
			p = inputPageCount - 1 - bookletPageNumber
		case 2:
			p = inputPageCount/2 - 1 - bookletPageNumber
		case 3:
			p = inputPageCount/2 + bookletPageNumber
		}
	}

	// Rotate bottom row of each output page by 180 degrees.
	var rotate bool
	if inputPageNr%4 >= 2 {
		rotate = true
	}
	return p, rotate
}

func nupLRTBOutputPageNr(positionNumber int, inputPageCount int, nup *model.NUp) (int, bool) {
	// move from left to right and then from top to bottom with no rotation
	var p int
	N := nup.N()
	bookletSheetSideNumber := positionNumber / N
	bookletSheetNumber := positionNumber / (2 * N)
	if bookletSheetSideNumber%2 == 0 {
		// front side
		if positionNumber%2 == 0 {
			// left side - count down from end
			p = inputPageCount - N*bookletSheetNumber - positionNumber%N
		} else {
			// right side - count up from start
			p = N*bookletSheetNumber + positionNumber%N
		}
	} else {
		// back side
		if positionNumber%2 == 0 {
			// left side - count up from start
			p = 2 + N*bookletSheetNumber + positionNumber%N
		} else {
			// right side - count down from end
			p = inputPageCount - N*bookletSheetNumber - positionNumber%N
		}
	}
	return p - 1, false // p is one-indexed and we want zero-indexed
}

func nup8OutputPageNr(positionNumber int, inputPageCount int, nup *model.NUp) (pageIdx int, rotate bool) {
	if nup.PageDim.Landscape() {
		positionNumber = landscapeToPortraitSheetPosition8up(positionNumber)
	}
	if nup.BookletBinding == model.ShortEdge {
		pageIdx, _ = nupLRTBOutputPageNr(positionNumber, inputPageCount, nup)
		if nup.PageDim.Landscape() {
			return pageIdx, true
		}
		return pageIdx, false
	}
	// else long edge
	// 8up sheet has four rows and two columns
	// but the spreads are NOT across the two columns - instead the spreads are rotated 90deg to fit in a portrait orientation on the sheet
	// rather than coding up an entire new imposition, we're going to use the left-right-top-bottom imposition as a base
	pageIdx, _ = nupLRTBOutputPageNr(n8upSpreadPosition(positionNumber), inputPageCount, nup)

	rotate = positionNumber%2 == 1 // rotate right column for portrait
	if nup.PageDim.Landscape() {
		rotate = !rotate // rotate bottom row for landscape
	}
	return pageIdx, rotate
}

func landscapeToPortraitSheetPosition8up(positionNumber int) int {
	// convert from landscape sheet position to portrait sheet position, by rotating counter clockwise
	return []int{6, 4, 2, 0, 7, 5, 3, 1}[positionNumber%8] + positionNumber/8*8
}
func n8upSpreadPosition(positionNumber int) int {
	// rotate the spreads (ie reorder) to fit on the sheet
	bookletSheetSideNumber := positionNumber / 8
	var out int
	switch bookletSheetSideNumber % 2 {
	case 0: // front side
		// rotate the block of four pages 90deg clockwise to go from portrait to landscape.             sequence=[1,3,0,2]
		// then because we are rotating the right side by 180deg - so need to change to those positions. sequence=[1,2,0,3]
		out = []int{1, 2, 0, 3}[positionNumber%4]
	case 1: // back side
		// rotate the block of four pages 90deg anti-clockwise to go from portrait to landscape.           sequence=[2,0,3,1]
		// then because we are rotating the *left* side by 180deg - so need to change to those positions. sequence=[3,0,2,1]
		// this is different from the front side because of the non-duplex sheet handling flip along the short edge
		out = []int{3, 0, 2, 1}[positionNumber%4]
	}
	return out + positionNumber/4*4
}

func nupPerfectBound(positionNumber int, inputPageCount int, nup *model.NUp) (int, bool) {
	var p int
	var rotate bool
	N := nup.N()
	twoN := N * 2

	bookletSheetSideNumber := positionNumber / N
	bookletSheetNumber := positionNumber / twoN
	if bookletSheetSideNumber%2 == 0 {
		// front side
		p = bookletSheetNumber*twoN + 2*(positionNumber%twoN) + 1
		rotate = N == 2
	} else {
		// back side
		p = bookletSheetNumber*twoN + 2*((positionNumber-N)%twoN) + 2
		if N == 4 || N == 6 || N == 8 {
			if nup.PageDim.Landscape() { // landscape pages on portrait sheets
				// flip top and bottom rows to account for landscape rotation and the page handling flip (short edge flip, no duplex)
				if positionNumber%N < N/2 { // top side
					p += N
				} else { // bottom side
					p -= N
				}
			} else { // portrait pages on portrait sheets
				// flip left and right columns to account for the page handling flip (short edge flip, no duplex)
				if positionNumber%2 == 0 { // left side
					p += 2
				} else { // right side
					p -= 2
				}
			}
		}
		// in these cases the page is rotated to fit onto the sheet
		// so we need to account for page handling flip (short edge flip, no duplex)
		rotate = (N == 4 && nup.PageDim.Landscape()) || (N == 8 && nup.PageDim.Portrait())
	}
	return p - 1, rotate // p is one-indexed and we want zero-indexed
}

func GetBookletOrdering(pages types.IntSet, nup *model.NUp, ordering orderingFn) []model.BookletPage {
	if ordering == nil {
		ordering = getBookletPageOrdering
	}
	pageNumbers := sortSelectedPages(pages)
	pageCount := len(pageNumbers)

	// A sheet of paper consists of 2 consecutive output pages.
	sheetPageCount := 2 * nup.N()

	// pageCount must be a multiple of the number of pages per sheet.
	// If not, we will insert blank pages at the end of the booklet.
	if pageCount%sheetPageCount != 0 {
		pageCount += sheetPageCount - pageCount%sheetPageCount
	}

	if nup.MultiFolio {
		bookletPages := make([]model.BookletPage, 0)
		// folioSize is the number of sheets - each "folio" has two sides and two pages per side
		nPagesPerSignature := nup.FolioSize * 4
		nSignaturesInBooklet := int(math.Ceil(float64(pageCount) / float64(nPagesPerSignature)))
		for j := 0; j < nSignaturesInBooklet; j++ {
			start := j * nPagesPerSignature
			stop := (j + 1) * nPagesPerSignature
			if stop > len(pageNumbers) {
				// last signature may be short
				stop = len(pageNumbers)
				nPagesPerSignature = pageCount - start
			}
			bookletPages = append(bookletPages, ordering(nup, pageNumbers[start:stop], nPagesPerSignature)...)
		}
		return bookletPages
	}
	return ordering(nup, pageNumbers, pageCount)
}

type orderingFn func(nup *model.NUp, pageNumbers []int, pageCount int) []model.BookletPage

func getBookletPageOrdering(nup *model.NUp, pageNumbers []int, pageCount int) []model.BookletPage {
	bookletPages := make([]model.BookletPage, pageCount)

	var pageNumberFn pageNumberFunction
	switch nup.BookletType {
	case model.Booklet, model.BookletAdvanced:
		switch nup.N() {
		case 2:
			pageNumberFn = nup2OutputPageNr
		case 4:
			pageNumberFn = nup4OutputPageNr
		case 6, 10:
			pageNumberFn = nupLRTBOutputPageNr
		case 8:
			pageNumberFn = nup8OutputPageNr
		}
	case model.BookletPerfectBound:
		pageNumberFn = nupPerfectBound
	}

	for i := 0; i < pageCount; i++ {
		pageIdx, rotate := pageNumberFn(i, pageCount, nup)
		if pageIdx >= len(pageNumbers) {
			bookletPages[i].IsBlank = true
			bookletPages[i].Number = pageIdx + pageNumbers[0] // typically pageIdx+1, but the pageNumbers[0] accounts for signatures
		} else {
			bookletPages[i].Number = pageNumbers[pageIdx]
		}

		bookletPages[i].Rotate = rotate
	}
	return bookletPages
}

func bookletPages(
	ctx *model.Context,
	selectedPages types.IntSet,
	nup *model.NUp,
	pagesDict types.Dict,
	pagesIndRef *types.IndirectRef,
	ordering orderingFn,
	rr []*types.Rectangle,
) (int, error) {
	var buf bytes.Buffer
	formsResDict := types.NewDict()
	if rr == nil {
		rr = nup.RectsForGrid()
	}
	j := 0

	for i, bp := range GetBookletOrdering(selectedPages, nup, ordering) {

		if i > 0 && i%len(rr) == 0 {
			// Wrap complete page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return 0, err
			}
			j++
			buf.Reset()
			formsResDict = types.NewDict()
		}

		rDest := rr[i%len(rr)]

		if bp.IsBlank {
			// This is an empty page at the end.
			if nup.BgColor != nil {
				draw.FillRectNoBorder(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		if err := ctx.NUpTilePDFBytesForPDF(bp.Number, formsResDict, &buf, rDest, nup, bp.Rotate); err != nil {
			return 0, err
		}
	}

	// Wrap incomplete booklet page.
	if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
		return 0, err
	}

	j++

	return j, nil
}

// BookletFromImages creates a booklet version of the image sequence represented by fileNames.
func BookletFromImages(ctx *model.Context, fileNames []string, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	// The order of images in fileNames corresponds to a desired booklet page sequence.
	selectedPages := types.IntSet{}
	for i := 1; i <= len(fileNames); i++ {
		selectedPages[i] = true
	}

	if nup.PageGrid {
		nup.PageDim.Width *= nup.Grid.Width
		nup.PageDim.Height *= nup.Grid.Height
	}

	xRefTable := ctx.XRefTable
	formsResDict := types.NewDict()
	var buf bytes.Buffer
	rr := nup.RectsForGrid()

	for i, bp := range GetBookletOrdering(selectedPages, nup, nil) {

		if i > 0 && i%len(rr) == 0 {

			// Wrap complete page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}

			buf.Reset()
			formsResDict = types.NewDict()
		}

		rDest := rr[i%len(rr)]

		if bp.IsBlank {
			// This is an empty page at the end of a booklet.
			if nup.BgColor != nil {
				draw.FillRectNoBorder(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		f, err := os.Open(fileNames[bp.Number-1])
		if err != nil {
			return err
		}

		imgIndRef, w, h, err := model.CreateImageResource(xRefTable, f)
		if err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}

		formIndRef, err := createNUpFormForImage(xRefTable, imgIndRef, w, h, i)
		if err != nil {
			return err
		}

		formResID := fmt.Sprintf("Fm%d", i)
		formsResDict.Insert(formResID, *formIndRef)

		// Append to content stream of booklet page i.
		model.NUpTilePDFBytes(&buf, types.RectForDim(float64(w), float64(h)), rr[i%len(rr)], formResID, nup, bp.Rotate)
	}

	// Wrap incomplete booklet page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

// BookletFromPDF creates a booklet version of the PDF represented by xRefTable.
func BookletFromPDF(ctx *model.Context, selectedPages types.IntSet, nup *model.NUp) error {
	n := int(nup.Grid.Width * nup.Grid.Height)
	if !types.IntMemberOf(n, NUpValuesForBooklets) {
		return fmt.Errorf("booklet must have nup in %v, got %d", NUpValuesForBooklets, n)
	}

	var mb *types.Rectangle

	if nup.PageDim == nil {
		nup.PageDim = types.PaperSize[nup.PageSize]
	}

	mb = types.RectForDim(nup.PageDim.Width, nup.PageDim.Height)

	pagesDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Pages"),
			"Count":    types.Integer(0),
			"MediaBox": mb.Array(),
		},
	)

	pagesIndRef, err := ctx.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	nup.PageDim = &types.Dim{Width: mb.Width(), Height: mb.Height()}

	pageCount, err := bookletPages(ctx, selectedPages, nup, pagesDict, pagesIndRef, nil, nil)
	if err != nil {
		return err
	}

	// Replace original pagesDict.
	rootDict, err := ctx.Catalog()
	if err != nil {
		return err
	}

	rootDict.Update("Pages", *pagesIndRef)

	ctx.PageCount = pageCount

	return nil
}

// BookletFromPDF creates a booklet version of the PDF represented by xRefTable.
func BookletFromPdfWithOrdering(ctx *model.Context, selectedPages types.IntSet, nup *model.NUp, ordering orderingFn, gridRects []*types.Rectangle) error {
	var mb *types.Rectangle
	if nup.PageDim == nil {
		nup.PageDim = types.PaperSize[nup.PageSize]
	}
	mb = types.RectForDim(nup.PageDim.Width, nup.PageDim.Height)

	pagesDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Pages"),
			"Count":    types.Integer(0),
			"MediaBox": mb.Array(),
		},
	)

	pagesIndRef, err := ctx.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	nup.PageDim = &types.Dim{Width: mb.Width(), Height: mb.Height()}

	pageCount, err := bookletPages(ctx, selectedPages, nup, pagesDict, pagesIndRef, ordering, gridRects)
	if err != nil {
		return err
	}

	// Replace original pagesDict.
	rootDict, err := ctx.Catalog()
	if err != nil {
		return err
	}

	rootDict.Update("Pages", *pagesIndRef)

	ctx.PageCount = pageCount

	return nil
}
