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

package api

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// ListImages returns a list of embedded images of rs.
func ListImages(rs io.ReadSeeker, selectedPages []string, conf *pdfcpu.Configuration) ([]string, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: ListImages: Please provide rs")
	}
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
		conf.Cmd = pdfcpu.LISTIMAGES
	}
	ctx, _, _, _, err := readValidateAndOptimize(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}
	if err := ctx.EnsurePageCount(); err != nil {
		return nil, err
	}
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return nil, err
	}

	return ctx.ListImages(pages)
}

// ListImagesFile returns a list of embedded images of inFile.
func ListImagesFile(inFiles []string, selectedPages []string, conf *pdfcpu.Configuration) ([]string, error) {
	if len(selectedPages) == 0 {
		log.CLI.Printf("pages: all\n")
	}
	ss := []string{}
	// Continue on error for file list.
	for _, fn := range inFiles {
		f, err := os.Open(fn)
		if err != nil {
			if len(inFiles) > 1 {
				ss = append(ss, fmt.Sprintf("\ncan't open %s: %v", fn, err))
				continue
			}
			return nil, err
		}
		defer f.Close()
		output, err := ListImages(f, selectedPages, conf)
		if err != nil {
			if len(inFiles) > 1 {
				ss = append(ss, fmt.Sprintf("\nproblem processing %s: %v", fn, err))
				continue
			}
			return nil, err
		}
		ss = append(ss, "\n"+fn)
		ss = append(ss, output...)
	}
	return ss, nil
}

func AlterImageFile(inFile, outFile, imageFile string, pageNumber int, imageID string, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
		// conf.Cmd = pdfcpu.TODO
	}
	var f1, f2, fImg *os.File
	f1, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f1.Close()

	if f2, err = os.Create(outFile); err != nil {
		return err
	}
	defer f2.Close()

	if fImg, err = os.Open(imageFile); err != nil {
		return err
	}
	defer fImg.Close()

	ctx, _, _, _, err := readValidateAndOptimize(f1, conf, time.Now())
	if err != nil {
		return err
	}
	if err := ctx.AlterImage(pageNumber, imageID, fImg); err != nil {
		return err
	}
	return WriteContext(ctx, f2)
}
