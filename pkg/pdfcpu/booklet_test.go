package pdfcpu

import (
	"fmt"
	"testing"
)

type pageOrderResults struct {
	pageCount         int
	expectedPageOrder []int
	papersize         string
}

var fourupTopFoldTestCases = []pageOrderResults{
	{
		pageCount: 8,
		expectedPageOrder: []int{
			8, 3, 1, 6,
			4, 7, 5, 2,
		},
		papersize: "A5",
	},
	{
		pageCount: 8,
		expectedPageOrder: []int{
			8, 3, 1, 6,
			2, 5, 7, 4,
		},
		papersize: "A5L",
	},
	{
		pageCount: 16,
		expectedPageOrder: []int{
			16, 3, 1, 14,
			4, 15, 13, 2,
			12, 7, 5, 10,
			8, 11, 9, 6,
		},
		papersize: "A5",
	},
}

func TestBookletPageOrder4UpTopFold(t *testing.T) {
	for _, test := range fourupTopFoldTestCases {
		nup := DefaultBookletConfig()
		err := ParseNUpDetails(fmt.Sprintf("papersize:%s", test.papersize), nup)
		if err != nil {
			t.Fatal(err)
		}
		pageNumbers := make([]int, test.pageCount)
		for i := 0; i < test.pageCount; i++ {
			pageNumbers[i] = i + 1
		}
		pageOrder := make([]int, test.pageCount)
		for i := range pageNumbers {
			p, _ := nup4TopFoldOutputPageNr(i, test.pageCount, pageNumbers, nup)
			pageOrder[i] = p
		}
		for i, expected := range test.expectedPageOrder {
			if pageOrder[i] != expected {
				t.Fatal("unexpected page order for test=", test, "\nexpected:", test.expectedPageOrder, "\ngot:", pageOrder)
			}
		}
	}
}
