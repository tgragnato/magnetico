package bencode

import (
	"reflect"
	"slices"
	"strings"
)

func getTag(st reflect.StructTag) tag {
	return parseTag(st.Get("bencode"))
}

type tag []string

func parseTag(tagStr string) tag {
	return strings.Split(tagStr, ",")
}

func (me tag) Ignore() bool {
	return me[0] == "-"
}

func (me tag) Key() string {
	return me[0]
}

func (me tag) HasOpt(opt string) bool {
	if len(me) < 1 {
		return false
	}
	return slices.Contains(me[1:], opt)
}

func (me tag) OmitEmpty() bool {
	return me.HasOpt("omitempty")
}

func (me tag) IgnoreUnmarshalTypeError() bool {
	return me.HasOpt("ignore_unmarshal_type_error")
}
