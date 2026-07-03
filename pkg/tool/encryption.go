package tool

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/anaskhan96/go-password-encoder"
	"golang.org/x/crypto/bcrypt"
)

var options = &password.Options{SaltLen: 16, Iterations: 100, KeyLen: 32, HashFunction: sha512.New}

func EncodePassWord(str string) string {
	salt, encodedPwd := password.Encode(str, options)
	newPassword := fmt.Sprintf("$pbkdf2-sha512$%s$%s", salt, encodedPwd)
	return newPassword
}

func VerifyPassWord(passwd, EncodePasswd string) bool {
	info := strings.Split(EncodePasswd, "$")
	if len(info) < 4 {
		return false
	}
	return password.Verify(passwd, info[2], info[3], options)
}

func MultiPasswordVerify(algo, salt, passwordText, hash string) bool {
	algo = strings.ToLower(strings.TrimSpace(algo))
	switch algo {
	case "md5":
		sum := md5.Sum([]byte(passwordText))
		return hex.EncodeToString(sum[:]) == hash
	case "sha256":
		sum := sha256.Sum256([]byte(passwordText))
		return hex.EncodeToString(sum[:]) == hash
	case "md5salt":
		sum := md5.Sum([]byte(passwordText + salt))
		return hex.EncodeToString(sum[:]) == hash
	case "sha256salt":
		sum := sha256.Sum256([]byte(passwordText + salt))
		return hex.EncodeToString(sum[:]) == hash
	case "bcrypt":
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(passwordText)) == nil
	default:
		return VerifyPassWord(passwordText, hash)
	}
}

func Md5Encode(str string, isUpper bool) string {
	sum := md5.Sum([]byte(str))
	res := hex.EncodeToString(sum[:])
	//转大写，strings.ToUpper(res)
	if isUpper {
		res = strings.ToUpper(res)
	}
	return res
}
