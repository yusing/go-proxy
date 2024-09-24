package functional

func FirstValueOf[KT comparable, VT any](m map[KT]VT) (_ VT, ok bool) {
	for _, v := range m {
		return v, true
	}
	return
}
