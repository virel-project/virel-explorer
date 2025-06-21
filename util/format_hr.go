package util

import "strconv"

func Unit(h float64) string {
	if h > 1000*1000*1000 {
		return strconv.FormatFloat(h/1000/1000/1000, 'f', 2, 64) + "M"
	} else if h > 1000*1000 {
		return strconv.FormatFloat(h/1000/1000, 'f', 2, 64) + "M"
	} else if h > 1000 {
		return strconv.FormatFloat(h/1000, 'f', 2, 64) + "K"
	}
	return strconv.FormatFloat(h, 'f', 2, 64)
}
