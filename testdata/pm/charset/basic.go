//go:build ignore

package main

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
)

const iroha = "いろはにほへとちりぬるをわかよたれそつねならむうゐのおくやまけふこえてあさきゆめみしゑひもせす"

func main() {
	testCJK()
	testUTF8Valid()
	testUTF8Invalid()
	testUTF16()
	testCharmaps()
	testLocaleCharset()
	testSpecialCases()
}

func testCJK() {
	for _, enc := range []encoding.Encoding{
		japanese.ShiftJIS,
		japanese.ISO2022JP,
		japanese.EUCJP,
		korean.EUCKR,
		simplifiedchinese.GB18030,
		simplifiedchinese.GBK,
		traditionalchinese.Big5,
	} {
		encoded, err := enc.NewEncoder().String(iroha)
		check(err)
		decoded, err := enc.NewDecoder().String(encoded)
		check(err)
		assert(decoded == iroha)
	}
}

func testUTF8Valid() {
	for _, s := range []string{
		"aiueo",
		"äöüß",
		"あいうえお",
		"あöあüあöあüあöあü",
		"asdf asdf asdfasd fasdfas dfあöあüa lksdjf alskdfj asalkdf kldfj asdあ aklsdjf asd",
		"\U0001F972",
	} {
		assert(utf8.ValidString(s))
		assert(strings.ToValidUTF8(s, "\uFFFD") == s)
	}
}

func testUTF8Invalid() {
	cases := map[string]string{
		"\xF8\x80\x80\x80\x80\x80":         "\uFFFD\uFFFD\uFFFD\uFFFD\uFFFD\uFFFD",
		"r\xC8sum\xC8s":                    "r\uFFFDsum\uFFFDs",
		"\x41\xC0\xAF\x41\xF4\x80\x80\x41": "A\uFFFDA\uFFFDA",
	}
	for input, want := range cases {
		assert(strings.ToValidUTF8(input, "\uFFFD") == want)
	}
}

func testUTF16() {
	be := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	le := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	mustDecode(be, "\x00H\x00e\x00l\x00l\x00o\x00,\x00 \x00w\x00o\x00r\x00l\x00d\x00!", "Hello, world!")
	mustDecode(le, "H\x00e\x00l\x00l\x00o\x00,\x00 \x00w\x00o\x00r\x00l\x00d\x00!\x00", "Hello, world!")
	mustDecode(be, "\xD8\x3E\xDD\x72", "\U0001F972")
	mustDecode(le, "\x3E\xD8\x72\xDD", "\U0001F972")
}

func testCharmaps() {
	const hungarian = "Nincsen apám, se anyám,\nse istenem, se hazám,\n"
	const german = "Wer reitet so spät durch Nacht und Wind?\nErlkönigs Töchter am düstern Ort?\n"
	mustRoundTrip(charmap.Windows1250, hungarian)
	mustRoundTrip(charmap.Windows1252, german)
	mustRoundTrip(charmap.ISO8859_2, hungarian)
}

func testLocaleCharset() {
	mustEncoding("euc-jp")
	mustEncoding("utf-8")
	assert(localeCharset("ja_JP.EUC_JP") == "euc-jp")
	assert(localeCharset("ja_JP.UTF-8") == "utf-8")
	assert(localeCharset("") == "utf-8")
}

func testSpecialCases() {
	sjisMinus, err := japanese.ShiftJIS.NewEncoder().String("\u2212")
	check(err)
	sjisFullwidthMinus, err := japanese.ShiftJIS.NewEncoder().String("\uFF0D")
	check(err)
	assert(sjisMinus == sjisFullwidthMinus)
	mustDecode(japanese.ISO2022JP, "\x1B\x24", "\uFFFD$")
	mustDecode(japanese.ISO2022JP, "\x1B\x28", "\uFFFD(")
	encoded, err := simplifiedchinese.GB18030.NewEncoder().String("\u0080")
	check(err)
	assert(encoded == "\x81\x30\x81\x30")
	mustDecode(simplifiedchinese.GB18030, "\x81\x30\x81\x30", "\u0080")
	mustDecode(simplifiedchinese.GB18030, "\x81\x3a", "\uFFFD:")
}

func mustRoundTrip(enc encoding.Encoding, s string) {
	encoded, err := enc.NewEncoder().String(s)
	check(err)
	decoded, err := enc.NewDecoder().String(encoded)
	check(err)
	assert(decoded == s)
}

func mustDecode(enc encoding.Encoding, input, want string) {
	got, err := enc.NewDecoder().String(input)
	check(err)
	assert(got == want)
}

func mustEncoding(label string) {
	_, err := htmlindex.Get(label)
	check(err)
}

func localeCharset(locale string) string {
	if strings.Contains(strings.ToLower(locale), "euc_jp") {
		return "euc-jp"
	}
	return "utf-8"
}

func assert(ok bool) {
	if !ok {
		panic("assertion failed")
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
