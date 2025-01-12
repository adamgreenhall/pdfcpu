/*
Copyright 2020 The pdf Authors.

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

package test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestExtractImages(t *testing.T) {
	msg := "TestExtractImages"
	// Extract images for all pages into outDir.
	for _, fn := range []string{"5116.DCT_Filter.pdf", "testImage.pdf", "go.pdf"} {
		// Test writing files
		fn = filepath.Join(inDir, fn)
		if err := api.ExtractImagesFile(fn, outDir, nil, nil); err != nil {
			t.Fatalf("%s %s: %v\n", msg, fn, err)
		}
	}
	// Extract images for inFile starting with page 1 into outDir.
	inFile := filepath.Join(inDir, "testImage.pdf")
	if err := api.ExtractImagesFile(inFile, outDir, []string{"1-"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func compare(t *testing.T, fn1, fn2 string) {

	f1, err := os.Open(fn1)
	if err != nil {
		t.Errorf("%s: %v", fn1, err)
		return
	}
	defer f1.Close()

	bb1, err := io.ReadAll(f1)
	if err != nil {
		t.Errorf("%s: %v", fn1, err)
		return
	}

	f2, err := os.Open(fn2)
	if err != nil {
		t.Errorf("%s: %v", fn2, err)
		return
	}
	defer f1.Close()

	bb2, err := io.ReadAll(f2)
	if err != nil {
		t.Errorf("%s: %v", fn2, err)
		return
	}

	if len(bb1) != len(bb2) {
		t.Errorf("%s <-> %s: length mismatch %d != %d", fn1, fn2, len(bb1), len(bb2))
		return
	}

	for i := 0; i < len(bb1); i++ {
		if bb1[i] != bb2[i] {
			t.Errorf("%s <-> %s: mismatch at %d, 0x%02x != 0x%02x\n", fn1, fn2, i, bb1[i], bb2[i])
			return
		}
	}

}

func TestExtractImagesSoftMasks(t *testing.T) {
	inFile := filepath.Join(inDir, "VectorApple.pdf")
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatal(err)
	}

	images := make(map[int]*types.StreamDict)
	for objId, obj := range ctx.XRefTable.Table {
		if obj != nil {
			if dict, ok := obj.Object.(types.StreamDict); ok {
				if subtype := dict.Dict.NameEntry("Subtype"); subtype != nil && *subtype == "Image" {
					images[objId] = &dict
				}
			}
		}
	}

	expected := map[int]string{
		36:  "VectorApple_36.tif",  // IndexedCMYK w/ softmask
		245: "VectorApple_245.tif", // DeviceCMYK w/ softmask
	}

	for objId, filename := range expected {
		sd := images[objId]

		if err := sd.Decode(); err != nil {
			t.Fatal(err)
		}

		tmpFileName := filepath.Join(outDir, filename)
		fmt.Printf("tmpFileName: %s\n", tmpFileName)

		// Write the image object (as TIFF file) to disk.
		// fn1 is the resulting fileName path including the suffix (aka filetype extension).
		fn1, err := pdfcpu.WriteImage(ctx.XRefTable, tmpFileName, sd, false, objId)
		if err != nil {
			t.Fatalf("err: %v\n", err)
		}

		fn2 := filepath.Join(resDir, filename)

		compare(t, fn1, fn2)
	}
}

func TestExtractImagesLowLevel(t *testing.T) {
	msg := "TestExtractImagesLowLevel"
	fileName := "testImage.pdf"
	inFile := filepath.Join(inDir, fileName)

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Optimize resource usage of this context.
	if err := api.OptimizeContext(ctx); err != nil {
		t.Fatalf("%s optimizeContext: %v\n", msg, err)
	}

	// Extract images for page 1.
	i := 1
	ii, err := pdfcpu.ExtractPageImages(ctx, i, false)
	if err != nil {
		t.Fatalf("%s extractPageFonts(%d): %v\n", msg, i, err)
	}

	baseFileName := strings.TrimSuffix(filepath.Base(fileName), ".pdf")

	// Process extracted images.
	for _, img := range ii {
		fn := filepath.Join(outDir, fmt.Sprintf("%s_%d_%s.%s", baseFileName, i, img.Name, img.FileType))
		if err := pdfcpu.WriteReader(fn, img); err != nil {
			t.Fatalf("%s write: %s", msg, fn)
		}
	}
}

func TestExtractFonts(t *testing.T) {
	msg := "TestExtractFonts"
	// Extract fonts for all pages into outDir.
	for _, fn := range []string{"5116.DCT_Filter.pdf", "testImage.pdf", "go.pdf"} {
		fn = filepath.Join(inDir, fn)
		if err := api.ExtractFontsFile(fn, outDir, nil, nil); err != nil {
			t.Fatalf("%s %s: %v\n", msg, fn, err)
		}
	}
	// Extract fonts for inFile for pages 1-3 into outDir.
	inFile := filepath.Join(inDir, "go.pdf")
	if err := api.ExtractFontsFile(inFile, outDir, []string{"1-3"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractFontsLowLevel(t *testing.T) {
	msg := "TestExtractFontsLowLevel"
	inFile := filepath.Join(inDir, "go.pdf")

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Optimize resource usage of this context.
	if err := api.OptimizeContext(ctx); err != nil {
		t.Fatalf("%s optimizeContext: %v\n", msg, err)
	}

	// Extract fonts for page 1.
	i := 1
	ff, err := pdfcpu.ExtractPageFonts(ctx, 1, types.IntSet{}, types.IntSet{})
	if err != nil {
		t.Fatalf("%s extractPageFonts(%d): %v\n", msg, i, err)
	}

	// Process extracted fonts.
	for _, f := range ff {
		fn := filepath.Join(outDir, fmt.Sprintf("%s.%s", f.Name, f.Type))
		if err := pdfcpu.WriteReader(fn, f); err != nil {
			t.Fatalf("%s write: %s", msg, fn)
		}
	}
}

func TestExtractPages(t *testing.T) {
	msg := "TestExtractPages"
	// Extract page #1 into outDir.
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	if err := api.ExtractPagesFile(inFile, outDir, []string{"1"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractPagesLowLevel(t *testing.T) {
	msg := "TestExtractPagesLowLevel"
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	outFile := "MyExtractedAndProcessedSinglePage.pdf"

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Extract page 1.
	i := 1

	r, err := api.ExtractPage(ctx, i)
	if err != nil {
		t.Fatalf("%s extractPage(%d): %v\n", msg, i, err)
	}

	if err := api.WritePage(r, outDir, outFile, i); err != nil {
		t.Fatalf("%s writePage(%d): %v\n", msg, i, err)
	}

}

func TestExtractContent(t *testing.T) {
	msg := "TestExtractContent"
	// Extract content of all pages into outDir.
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")
	if err := api.ExtractContentFile(inFile, outDir, nil, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractContentLowLevel(t *testing.T) {
	msg := "TestExtractContentLowLevel"
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Extract page content for page 2.
	i := 2
	r, err := pdfcpu.ExtractPageContent(ctx, i)
	if err != nil {
		t.Fatalf("%s extractPageContent(%d): %v\n", msg, i, err)
	}

	// Process page content.
	bb, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("%s readAll: %v\n", msg, err)
	}
	t.Logf("Page content (PDF-syntax) for page %d:\n%s", i, string(bb))
}

func TestExtractMetadata(t *testing.T) {
	msg := "TestExtractMetadata"
	// Extract all metadata into outDir.
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	if err := api.ExtractMetadataFile(inFile, outDir, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractMetadataLowLevel(t *testing.T) {
	msg := "TestExtractMedadataLowLevel"
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Extract all metadata.
	mm, err := pdfcpu.ExtractMetadata(ctx)
	if err != nil {
		t.Fatalf("%s ExtractMetadata: %v\n", msg, err)
	}

	// Process metadata.
	for _, md := range mm {
		bb, err := io.ReadAll(md)
		if err != nil {
			t.Fatalf("%s metadata readAll: %v\n", msg, err)
		}
		t.Logf("Metadata: objNr=%d parentDictObjNr=%d parentDictType=%s\n%s\n",
			md.ObjNr, md.ParentObjNr, md.ParentType, string(bb))
	}
}

func TestModifyPageContent(t *testing.T) {
	inFile := filepath.Join(inDir, "text-with-image.pdf")
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatal(err)
	}
	rexp := regexp.MustCompile(`(?mi)BT[\W\w]*ET\n`) // remove text object from BT to ET
	err = pdfcpu.ModifyPageContent(ctx, 1, func(c string) string {
		// log.Println("in\n", c)
		out := rexp.ReplaceAllString(c, "")
		// log.Println("out\n", out)
		return out
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := api.ValidateContext(ctx); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join("..", "..", "samples", "modify")
	outFile := filepath.Join(outDir, "removed-first-page-text.pdf")
	w, err := os.Create(outFile)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := api.Write(ctx, w, nil); err != nil {
		t.Fatal(err)
	}
}

func TestExtractImagePositions(t *testing.T) {
	cfg := model.NewDefaultConfiguration()
	cfg.Cmd = model.EXTRACTIMAGES
	for _, tc := range []struct {
		fnm      string
		pageNum  int
		expected []pdfcpu.ImagePositionsResult
	}{
		{"CenterOfWhy.pdf", 2, []pdfcpu.ImagePositionsResult{
			{ReferenceName: "Im0", Position: matrix.Matrix{{125.6502075, 0, 0}, {0, 148.0200043, 0}, {414, 571.9799957, 1}}},
			{ReferenceName: "Im1", Position: matrix.Matrix{{88.7711029, 0, 0}, {0, 120.1575928, 0}, {456.4575043, 367.2610016, 1}}},
		}},
	} {
		testName := fmt.Sprintf("%s:%d", tc.fnm, tc.pageNum)
		t.Run(testName, func(tt *testing.T) {
			f, err := os.Open(filepath.Join(inDir, tc.fnm))
			ok(tt, err)
			defer f.Close()
			ctx, err := api.ReadValidateAndOptimize(f, cfg)
			ok(tt, err)
			res, err := pdfcpu.ExtractImagePositions(ctx, tc.pageNum, true)
			ok(tt, err)
			for _, v := range tc.expected {
				found := false
				for _, r := range res {
					if r.ReferenceName == v.ReferenceName {
						if r.Position != v.Position {
							tt.Errorf("matrix not equal for image %s.\n expected: %s\n      got: %s", v.ReferenceName, v.Position, r.Position)
						}
						found = true
						break
					}
				}
				if !found {
					tt.Errorf("expected image name=%s in results %T", v.ReferenceName, res)
				}
			}
		})
	}
}
func ok(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
