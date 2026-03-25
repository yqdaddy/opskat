package backup_svc

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEncryptDecryptBackup(t *testing.T) {
	Convey("加密解密备份", t, func() {
		plainData := []byte(`{"version":1,"assets":[{"id":1,"name":"test"}]}`)
		password := "my-secret-password"

		Convey("正常加密解密往返", func() {
			encrypted, err := EncryptBackup(plainData, password)
			So(err, ShouldBeNil)
			So(encrypted, ShouldNotBeEmpty)

			decrypted, err := DecryptBackup(encrypted, password)
			So(err, ShouldBeNil)
			So(decrypted, ShouldResemble, plainData)
		})

		Convey("使用错误密码解密失败", func() {
			encrypted, err := EncryptBackup(plainData, password)
			So(err, ShouldBeNil)

			_, err = DecryptBackup(encrypted, "wrong-password")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "解密失败")
		})

		Convey("空数据加密解密往返", func() {
			emptyData := []byte(`{}`)
			encrypted, err := EncryptBackup(emptyData, password)
			So(err, ShouldBeNil)
			So(encrypted, ShouldNotBeEmpty)

			decrypted, err := DecryptBackup(encrypted, password)
			So(err, ShouldBeNil)
			So(decrypted, ShouldResemble, emptyData)
		})
	})
}

func TestIsEncryptedBackup(t *testing.T) {
	Convey("检测是否为加密备份", t, func() {
		Convey("加密数据返回 true", func() {
			plainData := []byte(`{"test":"data"}`)
			encrypted, err := EncryptBackup(plainData, "password")
			So(err, ShouldBeNil)
			So(IsEncryptedBackup(encrypted), ShouldBeTrue)
		})

		Convey("普通 JSON 返回 false", func() {
			So(IsEncryptedBackup([]byte(`{"format":"other","version":1}`)), ShouldBeFalse)
		})

		Convey("无效 JSON 返回 false", func() {
			So(IsEncryptedBackup([]byte(`not json`)), ShouldBeFalse)
		})

		Convey("空数据返回 false", func() {
			So(IsEncryptedBackup([]byte{}), ShouldBeFalse)
		})
	})
}

func TestDecryptBackupErrors(t *testing.T) {
	Convey("解密错误处理", t, func() {
		Convey("无效信封 JSON 返回错误", func() {
			_, err := DecryptBackup([]byte(`not valid json`), "password")
			So(err, ShouldNotBeNil)
		})

		Convey("不支持的版本返回错误", func() {
			envelope := EncryptedBackup{
				Format:  "opskat-encrypted-backup",
				Version: 99,
				KDF: KDFParams{
					Algorithm: "argon2id",
					Time:      3,
					Memory:    64 * 1024,
					Threads:   4,
					Salt:      "AAAAAAAAAAAAAAAAAAAAAA==",
				},
				Encryption: EncryptionParams{
					Algorithm: "aes-256-gcm",
					Nonce:     "AAAAAAAAAAAAAAAA",
				},
				Ciphertext: "AAAA",
			}
			data, err := json.Marshal(envelope)
			So(err, ShouldBeNil)

			_, err = DecryptBackup(data, "password")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "不支持的加密版本")
		})
	})
}
