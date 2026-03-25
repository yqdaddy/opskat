package sshpool

import (
	"bytes"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWriteReadFrame(t *testing.T) {
	Convey("WriteFrame 和 ReadFrame 往返测试", t, func() {
		Convey("正常负载", func() {
			var buf bytes.Buffer
			payload := []byte("hello, sshpool")
			err := WriteFrame(&buf, FrameStdout, payload)
			So(err, ShouldBeNil)

			ft, got, err := ReadFrame(&buf)
			So(err, ShouldBeNil)
			So(ft, ShouldEqual, FrameStdout)
			So(got, ShouldResemble, payload)
		})

		Convey("空负载", func() {
			var buf bytes.Buffer
			err := WriteFrame(&buf, FrameStdin, nil)
			So(err, ShouldBeNil)

			ft, got, err := ReadFrame(&buf)
			So(err, ShouldBeNil)
			So(ft, ShouldEqual, FrameStdin)
			So(got, ShouldBeNil)
		})

		Convey("负载超过最大值应返回错误", func() {
			var buf bytes.Buffer
			oversized := make([]byte, MaxFramePayload+1)
			err := WriteFrame(&buf, FrameStdout, oversized)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "payload too large")
		})
	})
}

func TestMultipleFramesInSequence(t *testing.T) {
	Convey("多帧连续读写", t, func() {
		var buf bytes.Buffer

		frames := []struct {
			ft      byte
			payload []byte
		}{
			{FrameStdout, []byte("first frame")},
			{FrameStderr, []byte("second frame")},
			{FrameStdin, []byte("third frame")},
		}

		for _, f := range frames {
			err := WriteFrame(&buf, f.ft, f.payload)
			So(err, ShouldBeNil)
		}

		for _, f := range frames {
			ft, payload, err := ReadFrame(&buf)
			So(err, ShouldBeNil)
			So(ft, ShouldEqual, f.ft)
			So(payload, ShouldResemble, f.payload)
		}
	})
}

func TestWriteReadExitCode(t *testing.T) {
	Convey("WriteExitCode 和 ParseExitCode 往返测试", t, func() {
		cases := []struct {
			desc string
			code int
		}{
			{"退出码 0", 0},
			{"退出码 127", 127},
			{"退出码 -1", -1},
		}

		for _, c := range cases {
			Convey(c.desc, func() {
				var buf bytes.Buffer
				err := WriteExitCode(&buf, c.code)
				So(err, ShouldBeNil)

				ft, payload, err := ReadFrame(&buf)
				So(err, ShouldBeNil)
				So(ft, ShouldEqual, FrameExitCode)

				code, err := ParseExitCode(payload)
				So(err, ShouldBeNil)
				So(code, ShouldEqual, c.code)
			})
		}

		Convey("无效负载长度应返回错误", func() {
			_, err := ParseExitCode([]byte{0x00, 0x01})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid exit code payload length")
		})
	})
}

func TestWriteReadResize(t *testing.T) {
	Convey("WriteResize 和 ParseResize 往返测试", t, func() {
		Convey("正常列数和行数", func() {
			var buf bytes.Buffer
			var cols uint16 = 220
			var rows uint16 = 50

			err := WriteResize(&buf, cols, rows)
			So(err, ShouldBeNil)

			ft, payload, err := ReadFrame(&buf)
			So(err, ShouldBeNil)
			So(ft, ShouldEqual, FrameResize)

			gotCols, gotRows, err := ParseResize(payload)
			So(err, ShouldBeNil)
			So(gotCols, ShouldEqual, cols)
			So(gotRows, ShouldEqual, rows)
		})

		Convey("无效负载长度应返回错误", func() {
			_, _, err := ParseResize([]byte{0x00, 0x01, 0x02})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid resize payload length")
		})
	})
}

func TestWriteError(t *testing.T) {
	Convey("WriteError 往返测试", t, func() {
		Convey("正常错误消息", func() {
			var buf bytes.Buffer
			msg := "connection refused"
			err := WriteError(&buf, msg)
			So(err, ShouldBeNil)

			ft, payload, err := ReadFrame(&buf)
			So(err, ShouldBeNil)
			So(ft, ShouldEqual, FrameError)
			So(string(payload), ShouldEqual, msg)
		})

		Convey("空错误消息", func() {
			var buf bytes.Buffer
			err := WriteError(&buf, "")
			So(err, ShouldBeNil)

			ft, payload, err := ReadFrame(&buf)
			So(err, ShouldBeNil)
			So(ft, ShouldEqual, FrameError)
			So(strings.TrimSpace(string(payload)), ShouldBeEmpty)
		})
	})
}
