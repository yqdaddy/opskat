package sortutil

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type testItem struct {
	id    int64
	order int
}

func getID(item testItem) int64  { return item.id }
func getOrder(item testItem) int { return item.order }

// noopUpdate 不记录调用的空 updateOrder
func noopUpdate(_ int64, _ int) error { return nil }

// recordUpdate 记录所有 updateOrder 调用
func recordUpdate(calls *[]struct{ id int64; order int }) func(int64, int) error {
	return func(id int64, order int) error {
		*calls = append(*calls, struct{ id int64; order int }{id, order})
		return nil
	}
}

func TestMoveItem(t *testing.T) {
	Convey("MoveItem 通用排序移动逻辑", t, func() {

		Convey("up - 与前一个元素交换顺序", func() {
			items := []testItem{
				{id: 1, order: 10},
				{id: 2, order: 20},
				{id: 3, order: 30},
			}
			var calls []struct{ id int64; order int }
			err := MoveItem(int64(2), "up", items, getID, getOrder, recordUpdate(&calls))
			So(err, ShouldBeNil)
			So(len(calls), ShouldEqual, 2)
			// item 2 gets item 1's order (10), item 1 gets item 2's order (20)
			So(calls[0].id, ShouldEqual, 2)
			So(calls[0].order, ShouldEqual, 10)
			So(calls[1].id, ShouldEqual, 1)
			So(calls[1].order, ShouldEqual, 20)
		})

		Convey("up - 已在第一位，无操作", func() {
			items := []testItem{
				{id: 1, order: 10},
				{id: 2, order: 20},
			}
			var calls []struct{ id int64; order int }
			err := MoveItem(int64(1), "up", items, getID, getOrder, recordUpdate(&calls))
			So(err, ShouldBeNil)
			So(len(calls), ShouldEqual, 0)
		})

		Convey("down - 与后一个元素交换顺序", func() {
			items := []testItem{
				{id: 1, order: 10},
				{id: 2, order: 20},
				{id: 3, order: 30},
			}
			var calls []struct{ id int64; order int }
			err := MoveItem(int64(2), "down", items, getID, getOrder, recordUpdate(&calls))
			So(err, ShouldBeNil)
			So(len(calls), ShouldEqual, 2)
			// item 2 gets item 3's order (30), item 3 gets item 2's order (20)
			So(calls[0].id, ShouldEqual, 2)
			So(calls[0].order, ShouldEqual, 30)
			So(calls[1].id, ShouldEqual, 3)
			So(calls[1].order, ShouldEqual, 20)
		})

		Convey("down - 已在最后一位，无操作", func() {
			items := []testItem{
				{id: 1, order: 10},
				{id: 2, order: 20},
			}
			var calls []struct{ id int64; order int }
			err := MoveItem(int64(2), "down", items, getID, getOrder, recordUpdate(&calls))
			So(err, ShouldBeNil)
			So(len(calls), ShouldEqual, 0)
		})

		Convey("top - 移动到第一位之前 (firstOrder-1)", func() {
			items := []testItem{
				{id: 1, order: 10},
				{id: 2, order: 20},
				{id: 3, order: 30},
			}
			var calls []struct{ id int64; order int }
			err := MoveItem(int64(3), "top", items, getID, getOrder, recordUpdate(&calls))
			So(err, ShouldBeNil)
			So(len(calls), ShouldEqual, 1)
			So(calls[0].id, ShouldEqual, 3)
			So(calls[0].order, ShouldEqual, 9) // firstOrder(10) - 1
		})

		Convey("top - 已在第一位，无操作", func() {
			items := []testItem{
				{id: 1, order: 10},
				{id: 2, order: 20},
			}
			var calls []struct{ id int64; order int }
			err := MoveItem(int64(1), "top", items, getID, getOrder, recordUpdate(&calls))
			So(err, ShouldBeNil)
			So(len(calls), ShouldEqual, 0)
		})

		Convey("item not found - 返回错误", func() {
			items := []testItem{
				{id: 1, order: 10},
				{id: 2, order: 20},
			}
			err := MoveItem(int64(99), "up", items, getID, getOrder, noopUpdate)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "item not found")
		})

		Convey("无效方向 - 返回错误", func() {
			items := []testItem{
				{id: 1, order: 10},
			}
			err := MoveItem(int64(1), "left", items, getID, getOrder, noopUpdate)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid direction")
		})

		Convey("up - 相同 order 值时自动递增以避免冲突", func() {
			items := []testItem{
				{id: 1, order: 10},
				{id: 2, order: 10}, // 与前一个 order 相同
			}
			var calls []struct{ id int64; order int }
			err := MoveItem(int64(2), "up", items, getID, getOrder, recordUpdate(&calls))
			So(err, ShouldBeNil)
			So(len(calls), ShouldEqual, 2)
			// curOrder becomes prevOrder+1 = 11 to avoid conflict
			So(calls[0].id, ShouldEqual, 2)
			So(calls[0].order, ShouldEqual, 10)
			So(calls[1].id, ShouldEqual, 1)
			So(calls[1].order, ShouldEqual, 11)
		})

		Convey("down - 相同 order 值时自动递增以避免冲突", func() {
			items := []testItem{
				{id: 1, order: 10},
				{id: 2, order: 10}, // 与后一个 order 相同
			}
			var calls []struct{ id int64; order int }
			err := MoveItem(int64(1), "down", items, getID, getOrder, recordUpdate(&calls))
			So(err, ShouldBeNil)
			So(len(calls), ShouldEqual, 2)
			// nextOrder becomes curOrder+1 = 11 to avoid conflict
			So(calls[0].id, ShouldEqual, 1)
			So(calls[0].order, ShouldEqual, 11)
			So(calls[1].id, ShouldEqual, 2)
			So(calls[1].order, ShouldEqual, 10)
		})

		Convey("updateOrder 错误传播", func() {
			items := []testItem{
				{id: 1, order: 10},
				{id: 2, order: 20},
			}
			errFail := fmt.Errorf("db error")
			failUpdate := func(_ int64, _ int) error { return errFail }

			Convey("up 方向第一次 updateOrder 失败时返回错误", func() {
				err := MoveItem(int64(2), "up", items, getID, getOrder, failUpdate)
				So(err, ShouldEqual, errFail)
			})

			Convey("down 方向第一次 updateOrder 失败时返回错误", func() {
				err := MoveItem(int64(1), "down", items, getID, getOrder, failUpdate)
				So(err, ShouldEqual, errFail)
			})

			Convey("top 方向 updateOrder 失败时返回错误", func() {
				err := MoveItem(int64(2), "top", items, getID, getOrder, failUpdate)
				So(err, ShouldEqual, errFail)
			})
		})
	})
}
