package secure

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewBytes(t *testing.T) {
	Convey("NewBytes", t, func() {
		Convey("应复制数据而非引用", func() {
			original := []byte("sensitive")
			sb := NewBytes(original)

			// 修改原始数据不影响 Bytes 内部数据
			original[0] = 'X'
			So(sb.Bytes()[0], ShouldEqual, byte('s'))
		})

		Convey("正常读取数据", func() {
			data := []byte("hello")
			sb := NewBytes(data)
			So(sb.Bytes(), ShouldResemble, []byte("hello"))
		})

		Convey("String 方法返回正确字符串", func() {
			sb := NewBytes([]byte("world"))
			So(sb.String(), ShouldEqual, "world")
		})

		Convey("空数据", func() {
			sb := NewBytes([]byte{})
			So(sb.Bytes(), ShouldResemble, []byte{})
			So(sb.String(), ShouldEqual, "")
		})
	})
}

func TestNewBytesFromString(t *testing.T) {
	Convey("NewBytesFromString", t, func() {
		Convey("从字符串创建安全字节封装", func() {
			sb := NewBytesFromString("secret")
			So(sb.Bytes(), ShouldResemble, []byte("secret"))
			So(sb.String(), ShouldEqual, "secret")
		})

		Convey("空字符串", func() {
			sb := NewBytesFromString("")
			So(sb.Bytes(), ShouldResemble, []byte{})
			So(sb.String(), ShouldEqual, "")
		})
	})
}

func TestBytesZero(t *testing.T) {
	Convey("Zero", t, func() {
		Convey("清零后数据应不可访问（返回 nil）", func() {
			sb := NewBytes([]byte("sensitive"))
			So(sb.IsZeroed(), ShouldBeFalse)

			sb.Zero()

			So(sb.Bytes(), ShouldBeNil)
			So(sb.String(), ShouldEqual, "")
		})

		Convey("清零后 IsZeroed 应返回 true", func() {
			sb := NewBytesFromString("data")
			sb.Zero()
			So(sb.IsZeroed(), ShouldBeTrue)
		})

		Convey("Zero 应是幂等的（多次调用不 panic）", func() {
			sb := NewBytes([]byte("idempotent"))
			sb.Zero()
			So(func() { sb.Zero() }, ShouldNotPanic)
			So(sb.IsZeroed(), ShouldBeTrue)
			So(sb.Bytes(), ShouldBeNil)
		})
	})
}

func TestZeroSlice(t *testing.T) {
	Convey("ZeroSlice", t, func() {
		Convey("所有字节应变为 0", func() {
			data := []byte{1, 2, 3, 4, 5}
			ZeroSlice(data)
			for _, b := range data {
				So(b, ShouldEqual, 0)
			}
		})

		Convey("空切片不 panic", func() {
			So(func() { ZeroSlice([]byte{}) }, ShouldNotPanic)
		})
	})
}
