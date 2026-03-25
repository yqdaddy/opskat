package jsonfield

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type testStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestUnmarshal(t *testing.T) {
	Convey("Unmarshal 测试", t, func() {
		Convey("正常 JSON 解析成功", func() {
			result, err := Unmarshal[testStruct](`{"name":"foo","value":42}`, "测试")
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			So(result.Name, ShouldEqual, "foo")
			So(result.Value, ShouldEqual, 42)
		})

		Convey("空字符串返回错误", func() {
			result, err := Unmarshal[testStruct]("", "测试")
			So(err, ShouldNotBeNil)
			So(result, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "测试为空")
		})

		Convey("非法 JSON 返回错误", func() {
			result, err := Unmarshal[testStruct]("not-json", "测试")
			So(err, ShouldNotBeNil)
			So(result, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "解析测试失败")
		})
	})
}

func TestUnmarshalOrDefault(t *testing.T) {
	Convey("UnmarshalOrDefault 测试", t, func() {
		Convey("空字符串返回零值", func() {
			result, err := UnmarshalOrDefault[testStruct]("", "测试")
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			So(result.Name, ShouldEqual, "")
			So(result.Value, ShouldEqual, 0)
		})

		Convey("正常 JSON 解析成功", func() {
			result, err := UnmarshalOrDefault[testStruct](`{"name":"bar","value":7}`, "测试")
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			So(result.Name, ShouldEqual, "bar")
			So(result.Value, ShouldEqual, 7)
		})

		Convey("非法 JSON 返回错误", func() {
			result, err := UnmarshalOrDefault[testStruct]("{bad json}", "测试")
			So(err, ShouldNotBeNil)
			So(result, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "解析测试失败")
		})
	})
}

func TestMarshal(t *testing.T) {
	Convey("Marshal 测试", t, func() {
		Convey("正常序列化成功", func() {
			v := &testStruct{Name: "baz", Value: 99}
			result, err := Marshal(v, "测试")
			So(err, ShouldBeNil)
			So(result, ShouldContainSubstring, `"name":"baz"`)
			So(result, ShouldContainSubstring, `"value":99`)
		})
	})
}

func TestMarshalOrClear(t *testing.T) {
	Convey("MarshalOrClear 测试", t, func() {
		isEmpty := func(v *testStruct) bool {
			return v.Name == "" && v.Value == 0
		}

		Convey("nil 返回空字符串", func() {
			result, err := MarshalOrClear((*testStruct)(nil), isEmpty, "测试")
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "")
		})

		Convey("isEmpty 返回 true 时返回空字符串", func() {
			v := &testStruct{}
			result, err := MarshalOrClear(v, isEmpty, "测试")
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "")
		})

		Convey("正常值序列化成功", func() {
			v := &testStruct{Name: "qux", Value: 3}
			result, err := MarshalOrClear(v, isEmpty, "测试")
			So(err, ShouldBeNil)
			So(result, ShouldContainSubstring, `"name":"qux"`)
			So(result, ShouldContainSubstring, `"value":3`)
		})
	})
}
