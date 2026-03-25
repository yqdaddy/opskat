package credential_entity

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestValidate(t *testing.T) {
	Convey("Validate 校验", t, func() {
		Convey("名称为空时返回错误", func() {
			c := &Credential{
				Name:     "",
				Type:     TypePassword,
				Password: "secret",
			}
			err := c.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "凭证名称不能为空")
		})

		Convey("密码类型", func() {
			Convey("密码为空时返回错误", func() {
				c := &Credential{
					Name:     "test",
					Type:     TypePassword,
					Password: "",
				}
				err := c.Validate()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "密码不能为空")
			})

			Convey("密码不为空时校验通过", func() {
				c := &Credential{
					Name:     "test",
					Type:     TypePassword,
					Password: "mypassword",
				}
				err := c.Validate()
				So(err, ShouldBeNil)
			})
		})

		Convey("SSH 密钥类型", func() {
			Convey("私钥为空时返回错误", func() {
				c := &Credential{
					Name:      "test",
					Type:      TypeSSHKey,
					PrivateKey: "",
					PublicKey:  "pubkey",
					KeyType:    KeyTypeED25519,
				}
				err := c.Validate()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "私钥不能为空")
			})

			Convey("公钥为空时返回错误", func() {
				c := &Credential{
					Name:      "test",
					Type:      TypeSSHKey,
					PrivateKey: "privkey",
					PublicKey:  "",
					KeyType:    KeyTypeED25519,
				}
				err := c.Validate()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "公钥不能为空")
			})

			Convey("不支持的密钥类型返回错误", func() {
				c := &Credential{
					Name:      "test",
					Type:      TypeSSHKey,
					PrivateKey: "privkey",
					PublicKey:  "pubkey",
					KeyType:    "dsa",
				}
				err := c.Validate()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "不支持的密钥类型")
			})

			Convey("ed25519 密钥类型校验通过", func() {
				c := &Credential{
					Name:      "test",
					Type:      TypeSSHKey,
					PrivateKey: "privkey",
					PublicKey:  "pubkey",
					KeyType:    KeyTypeED25519,
				}
				err := c.Validate()
				So(err, ShouldBeNil)
			})

			Convey("ecdsa 密钥类型校验通过", func() {
				c := &Credential{
					Name:      "test",
					Type:      TypeSSHKey,
					PrivateKey: "privkey",
					PublicKey:  "pubkey",
					KeyType:    KeyTypeECDSA,
				}
				err := c.Validate()
				So(err, ShouldBeNil)
			})

			Convey("rsa 密钥类型校验通过", func() {
				c := &Credential{
					Name:      "test",
					Type:      TypeSSHKey,
					PrivateKey: "privkey",
					PublicKey:  "pubkey",
					KeyType:    KeyTypeRSA,
				}
				err := c.Validate()
				So(err, ShouldBeNil)
			})
		})

		Convey("不支持的凭证类型返回错误", func() {
			c := &Credential{
				Name: "test",
				Type: "unknown",
			}
			err := c.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "不支持的凭证类型")
		})
	})
}

func TestIsPassword(t *testing.T) {
	Convey("IsPassword 类型判断", t, func() {
		Convey("密码类型返回 true", func() {
			c := &Credential{Type: TypePassword}
			So(c.IsPassword(), ShouldBeTrue)
		})

		Convey("SSH 密钥类型返回 false", func() {
			c := &Credential{Type: TypeSSHKey}
			So(c.IsPassword(), ShouldBeFalse)
		})

		Convey("其他类型返回 false", func() {
			c := &Credential{Type: "unknown"}
			So(c.IsPassword(), ShouldBeFalse)
		})
	})
}

func TestIsSSHKey(t *testing.T) {
	Convey("IsSSHKey 类型判断", t, func() {
		Convey("SSH 密钥类型返回 true", func() {
			c := &Credential{Type: TypeSSHKey}
			So(c.IsSSHKey(), ShouldBeTrue)
		})

		Convey("密码类型返回 false", func() {
			c := &Credential{Type: TypePassword}
			So(c.IsSSHKey(), ShouldBeFalse)
		})

		Convey("其他类型返回 false", func() {
			c := &Credential{Type: "unknown"}
			So(c.IsSSHKey(), ShouldBeFalse)
		})
	})
}
