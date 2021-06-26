package pdfcpu

import (
	"fmt"
	"testing"
)

type pageOrderResults struct {
	pageCount         int
	expectedPageOrder []int
	papersize         string
	bookletType       string
}

var bookletTestCases = []pageOrderResults{
	// classic (booklet) test cases
	{
		pageCount: 8,
		expectedPageOrder: []int{
			8, 1, 5, 4,
			2, 7, 3, 6,
		},
		papersize:   "A5", // portrait, long-edge binding
		bookletType: "booklet",
	},
	{
		pageCount: 8,
		expectedPageOrder: []int{
			8, 1, 5, 4,
			6, 3, 7, 2, // this is 180degrees flipped from the portrait layout (because of differences in how duplexing works)
		},
		papersize:   "A5L", // landscape, short-edge binding
		bookletType: "booklet",
	},
	// topfold test cases
	{
		pageCount: 8,
		expectedPageOrder: []int{
			8, 3, 1, 6,
			4, 7, 5, 2,
		},
		papersize:   "A5", // portrait, short-edge binding
		bookletType: "booklet-topfold",
	},
	{
		pageCount: 8,
		expectedPageOrder: []int{
			8, 3, 1, 6,
			2, 5, 7, 4,
		},
		papersize:   "A5L", // landscape, long-edge binding
		bookletType: "booklet-topfold",
	},
	{
		pageCount: 16,
		expectedPageOrder: []int{
			16, 3, 1, 14,
			4, 15, 13, 2,
			12, 7, 5, 10,
			8, 11, 9, 6,
		},
		papersize:   "A5", // portrait, short-edge binding
		bookletType: "booklet-topfold",
	},
}

func TestBookletPageOrder(t *testing.T) {
	for _, test := range bookletTestCases {
		nup, err := PDFBookletConfig(4, fmt.Sprintf("papersize:%s, btype:%s", test.papersize, test.bookletType))
		if err != nil {
			t.Fatal(err)
		}
		pageNumbers := make(map[int]bool)
		for i := 0; i < test.pageCount; i++ {
			pageNumbers[i+1] = true
		}
		pageOrder := make([]int, test.pageCount)
		for i, p := range sortSelectedPagesForBooklet(pageNumbers, nup) {
			pageOrder[i] = p.number
		}
		for i, expected := range test.expectedPageOrder {
			if pageOrder[i] != expected {
				t.Fatal("unexpected page order for test=", test, "\nexpected:", test.expectedPageOrder, "\ngot:", pageOrder)
			}
		}
	}
}
