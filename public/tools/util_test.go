package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/eryajf/go-ldap-admin/config"
)

func TestGenPass(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()
	config.InitConfig()
	fmt.Printf("密码为：%s\n", NewGenPasswd("123456"))
	// err := ComparePasswd("$2a$10$Fy8p0nCixgWKzLfO3SgdhOzAF7YolSt6dHj6QidDGYlzLJDpniXB6", "123456")
	// if err != nil {
	// 	fmt.Printf("密码错误：%s\n", err)
	// }
}

func TestArrUintCmp(t *testing.T) {
	a := []uint{1, 2, 3, 4, 6, 9}
	b := []uint{1, 2, 3, 4, 5, 6, 7}
	c, d := ArrUintCmp(a, b)
	fmt.Printf("%v\n", c)
	fmt.Printf("%v\n", d)
}

func TestSliceToString(t *testing.T) {
	a := []uint{1}
	fmt.Printf("%s\n", SliceToString(a, ","))
}

func TestEncodePass(t *testing.T) {
	// to encode a password into ssha
	hashed := EncodePass([]byte("testpass"))
	fmt.Println(string(hashed))
	// to validate a password against saved hash.
	if Matches([]byte(hashed), []byte("testpass")) {
		fmt.Println("Its a match.")
	} else {
		fmt.Println("its not match")
	}
}

func TestConvertToUIDShort(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"王建国", "jgwang"},   // 3-char: surname wang, given jian+guo → jg
		{"张三", "szhang"},    // 2-char: surname zhang, given san → s
		{"欧阳娜娜", "nnouyang"}, // compound surname: ouyang, given na+na → nn
		{"芳", "fang"},        // single char — surname only, no prefix
		{"李", "li"},          // single char
		{"john", "ohnj"},     // ASCII: j=surname full, o+h+n → given initials
	}
	for _, c := range cases {
		got := ConvertToUIDShort(c.input)
		if got != c.want {
			t.Errorf("ConvertToUIDShort(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
