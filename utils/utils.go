package utils

func UnlessNilThenPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func MapMerge(dst map[string]string,
	src map[string]string, overwrite bool) map[string]string {
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
