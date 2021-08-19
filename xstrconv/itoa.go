package xstrconv

const fastSmalls = true // enable fast path for small integers

// Itoa is equivalent to FormatInt(int64(i), 10).
func Itoa(i int) string {
	return FormatInt(int64(i), 10)
}

func FormatInt(i int64, base int) string {
	if fastSmalls && 0 <= i && i < nSmalls && base == 10 {
		return small(int(i))
	}
	_, s := formatBits(nil, uint64(i), base, i < 0, false)
	return s
}

// small returns the string for an i with 0 <= i < nSmalls.
// 0<=i<100时
func small(i int) string {
	// [0, 10)
	if i < 10 {
		return digits[i : i+1]
	}
	// [10, 100)
	return smallsString[i*2 : i*2+2]
}

const nSmalls = 100

const smallsString = "00010203040506070809" +
	"10111213141516171819" +
	"20212223242526272829" +
	"30313233343536373839" +
	"40414243444546474849" +
	"50515253545556575859" +
	"60616263646566676869" +
	"70717273747576777879" +
	"80818283848586878889" +
	"90919293949596979899"

const digits = "0123456789abcdefghijklmnopqrstuvwxyz"

// formatBits computes the string representation of u in the given base.
// If neg is set, u is treated as negative int64 value. If append_ is
// set, the string is appended to dst and the resulting byte slice is
// returned as the first result value; otherwise the string is returned
// as the second result value.
//
func formatBits(dst []byte, u uint64, base int, neg, append_ bool) (d []byte, s string) {
	if base < 2 || base > len(digits) {
		panic("strconv: illegal AppendInt/FormatInt base")
	}
	// 2 <= base && base <= len(digits), 必须为2到36之间

	var a [64 + 1]byte // +1 for sign of 64bit value in base 2
	i := len(a)        // 65

	if neg {
		u = -u
	}

	if base == 10 {
		// u guaranteed to fit into a uint
		us := uint(u)
		for us >= 100 {
			is := us % 100 * 2
			us /= 100
			i -= 2
			a[i+1] = smallsString[is+1]
			a[i+0] = smallsString[is+0]
		}

		// us < 100
		// 这里挺妙的，如果我写我就想不到
		is := us * 2
		i--
		a[i] = smallsString[is+1]
		if us >= 10 {
			i--
			a[i] = smallsString[is]
		}
	} else {
		panic("todo")
	}

	// add sign, if any
	if neg {
		i--
		a[i] = '-'
	}
	if append_ {
		d = append(dst, a[i:]...)
		return
	}
	s = string(a[i:])
	return
}
