package api

import (
	"log"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestResizeOntoPage(t *testing.T) {
	inFile := filepath.Join("..", "testdata", "Walden.pdf")
	ctx, err := ReadContextFile(inFile)
	if err != nil {
		log.Fatal(t, err)
	}
	if err := pdfcpu.ResizeOntoPage(ctx, pdfcpu.ResizeParams{
		// original size: 595.28 x 841.89 points
		ContentDim: types.Dim{Width: 500, Height: 500 / 0.7070707},
		PageDim:    types.Dim{Width: 700, Height: 850},
		PerPageParams: map[int]pdfcpu.ResizeParamsPage{
			1: {Anchor: types.Left},
			2: {Anchor: types.Right},
		},
	}, nil); err != nil {
		log.Fatal(t, err)
	}
	if err := WriteContextFile(ctx, filepath.Join("..", "samples", "resize", "resizeOntoPage.pdf")); err != nil {
		log.Fatal(t, err)
	}
}
