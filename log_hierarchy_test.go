package log

import (
	"testing"

	apex "github.com/apex/log"
	"github.com/apex/log/handlers/memory"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHierarchy(t *testing.T) {
	c := Config{
		Level:   "normal",
		Handler: "memory",
		Named: map[string]*Config{
			"/test/level/debug": {
				Level: "debug",
			},
			"/test/level/warn": {
				Level: "warn",
			},
		},
	}
	SetDefault(&c)
	handler := def.Handler().(*memory.Handler)

	Convey("Given a hierarchical log configuration", t, func() {
		clearEntries(handler)

		Convey("getting a logger with empty name returns the root logger", func() {
			l := Get("")
			So(l, ShouldNotBeNil)

			l.Info("info")
			l.Debug("debug")

			So(len(handler.Entries), ShouldEqual, 1)
			So(handler.Entries[0].Message, ShouldEqual, "info")
			So(handler.Entries[0].Fields.Get("logger"), ShouldBeNil)
		})

		Convey("getting a named logger returns a correctly configured instance", func() {
			l := Get("/test/level/debug")
			So(l, ShouldNotBeNil)

			l.Info("info")
			l.Debug("debug")

			So(len(handler.Entries), ShouldEqual, 2)
			So(handler.Entries[0].Message, ShouldEqual, "info")
			So(handler.Entries[0].Fields.Get("logger"), ShouldEqual, "/test/level/debug")
			So(handler.Entries[1].Message, ShouldEqual, "debug")
			So(handler.Entries[1].Fields.Get("logger"), ShouldEqual, "/test/level/debug")

			Convey("getting the same named logger returns the same logger instance", func() {
				l2 := Get("/test/level/debug")
				So(l == l2, ShouldBeTrue)
			})

			Convey("getting another named logger returns another instance", func() {
				l2 := Get("/test/level/warn")
				So(l != l2, ShouldBeTrue)

				clearEntries(handler)
				l2.Info("info")
				l2.Debug("debug")

				So(len(handler.Entries), ShouldEqual, 0)

				l2.Warn("warn")

				So(len(handler.Entries), ShouldEqual, 1)
				So(handler.Entries[0].Message, ShouldEqual, "warn")
				So(handler.Entries[0].Fields.Get("logger"), ShouldBeNil)

				Convey("getting a sub-logger without own config returns a custom logger, configured like the parent logger, but with adapted logger field", func() {
					l3 := Get("/test/level/debug/sub/sub")
					So(l != l3, ShouldBeTrue)

					clearEntries(handler)
					l3.Debug("debug")

					So(len(handler.Entries), ShouldEqual, 1)
					So(handler.Entries[0].Message, ShouldEqual, "debug")
					So(handler.Entries[0].Fields.Get("logger"), ShouldEqual, "/test/level/debug/sub/sub")
				})
			})
		})
	})
}
func clearEntries(handler *memory.Handler) {
	handler.Entries = make([]*apex.Entry, 0)
}
