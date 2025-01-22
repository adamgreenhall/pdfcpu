package pdfcpu_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func TestInsertImageRefIntoPdf(t *testing.T) {
	fnmIn := "../testdata/mountain.pdf"
	fnmImg := "../testdata/resources/snow.jpg"
	ctx, err := api.ReadContextFile(fnmIn)
	ok(t, err)
	ctx.Conf.OptimizeResourceDicts = true
	ok(t, api.OptimizeContext(ctx))
	fileImg, err := os.Open(fnmImg)
	ok(t, err)
	defer fileImg.Close()
	objNum, _, err := pdfcpu.InsertImageRefIntoPdf(ctx, fileImg, []int{1})
	ok(t, err)
	// to see if this is working, append page content and write pdf to file
	ok(t, appendPageContent(ctx, 1, fmt.Sprintf("\nq 600 0 0 300 0 0 cm /Im%d Do Q\n", objNum)))
	ok(t, api.ValidateContext(ctx))
	ok(t, api.WriteContextFile(ctx, "../samples/images/TestInsertImageRefIntoPdf.pdf"))
}

func ok(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func appendPageContent(ctx *model.Context, pageNum int, newContentAppend string) error {
	d, _, _, err := ctx.PageDict(pageNum, false)
	if err != nil {
		return err
	}
	bb, err := ctx.PageContent(d)
	if err != nil {
		return err
	}
	bb = append(bb, []byte(newContentAppend)...)
	sd, _ := ctx.NewStreamDictForBuf(bb)
	sd.Encode()

	oRef, _ := d.Find("Contents")
	if oRef == nil {
		return model.ErrNoContent
	}
	oInd, ok := oRef.(types.IndirectRef)
	if !ok {
		return errors.New("pdfcpu: couldnt get contents ref")
	}

	// overwrite content ref object
	entry, ok := ctx.FindTableEntry(oInd.ObjectNumber.Value(), 0)
	if !ok {
		return errors.Errorf("pdfcpu: invalid objNr=%d", oInd.ObjectNumber.Value())
	}
	entry.Object = *sd
	return nil
}
