package utils

func UnlessNilThenPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func MapMerge(dst map[string]interface{},
	src map[string]interface{}, overwrite bool) map[string]interface{} {
	if src == nil {
		return dst
	}

	for k, v := range src {
		if _, exists := dst[k]; exists && !overwrite {
			continue
		}
		dst[k] = v
	}

	return dst
}

func NS2MS(ns int64) float64 {
	return float64(ns)/float64(1000000)
}
