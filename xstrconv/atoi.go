package xstrconv

import "errors"

const intSize = 32 << (^uint(0) >> 63) // 64
const IntSize = intSize

// Atoi is equivalent to ParseInt(s, 10, 0), converted to type int.
func Atoi(s string) (int, error) {
	const fnAtoi = "Atoi"

	sLen := len(s)
	if intSize == 32 && (0 < sLen && sLen < 10) ||
		intSize == 64 && (0 < sLen && sLen < 19) { // maxInt64 9223372036854775807
		// Fast path for small integers that fit int type.
		s0 := s
		if s[0] == '-' || s[0] == '+' {
			s = s[1:]
			if len(s) < 1 {
				return 0, &NumError{fnAtoi, s0, ErrSyntax}
			}
		}

		n := 0
		for _, ch := range []byte(s) {
			ch -= '0'
			if ch > 9 {
				return 0, &NumError{fnAtoi, s0, ErrSyntax}
			}
			n = n*10 + int(ch)
		}
		if s0[0] == '-' {
			n = -n
		}
		return n, nil
	}
	return 0, errors.New("todo")
	// Slow path for invalid, big, or underscored integers.
	// i64, err := ParseInt(s, 10, 0)
	// if nerr, ok := err.(*NumError); ok {
	// 	nerr.Func = fnAtoi
	// }
	// return int(i64), err
}
