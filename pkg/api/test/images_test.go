package test

import (
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func TestAlterPDFImage(t *testing.T) {
	inFile := path.Join(inDir, "testImage.pdf")
	outFile := path.Join(outDir, "testImage.altered.pdf")
	imagesPathPrefix := "extractImage"
	f, err := os.Open(inFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := api.ExtractImages(f, []string{"2"}, pdfcpu.WriteImageToDisk(outDir, imagesPathPrefix), nil); err != nil {
		t.Fatal(err)
	}
	imageFilenames, err := filepath.Glob(path.Join(outDir, imagesPathPrefix+"*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(imageFilenames) != 1 {
		t.Errorf("more than one image extracted")
	}
	imgFilename := imageFilenames[0]
	parts := strings.Split(strings.Trim(path.Base(imgFilename), path.Ext(imgFilename)), "_")
	imageID := parts[len(parts)-1]

	// alter image to be grayscale
	fImg, err := os.Open(imgFilename)
	if err != nil {
		t.Fatal(err)
	}
	defer fImg.Close()
	img, _, err := image.Decode(fImg)
	if err != nil {
		t.Fatal(err)
	}
	imgGray := image.NewGray(img.Bounds())
	draw.Draw(imgGray, imgGray.Rect, img, image.Point{}, draw.Src)
	imgFilenameAltered := strings.Trim(imgFilename, ".jpg") + ".altered.jpg"
	fImgAlt, err := os.Create(imgFilenameAltered)
	if err != nil {
		t.Fatal(err)
	}
	// TODO: png vs jpeg vs other formats
	if err := jpeg.Encode(fImgAlt, imgGray, nil); err != nil {
		t.Fatal(err)
	}
	if err := fImgAlt.Close(); err != nil {
		t.Fatal(err)
	}

	if err := api.AlterImageFile(inFile, outFile, imgFilenameAltered, 2, imageID, nil); err != nil {
		t.Fatal(err)
	}
	log.Println(outFile)
	log.Println("ok")
}
