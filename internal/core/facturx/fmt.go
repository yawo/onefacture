package facturx

import "strconv"

func ftoa(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
func itoa(v int) string     { return strconv.Itoa(v) }
