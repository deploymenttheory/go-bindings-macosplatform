//go:build darwin

package purego

import "github.com/ebitengine/purego/objc"

var (
	selCount          = objc.RegisterName("count")
	selObjectAtIndex  = objc.RegisterName("objectAtIndex:")
	selAddObject      = objc.RegisterName("addObject:")
	selArray          = objc.RegisterName("array")
	clsNSMutableArray = objc.GetClass("NSMutableArray")
)

// NSArrayCount returns the number of elements in an NSArray. It sends -count
// directly to the array id, sidestepping the typed-NSArray wrapper whose value
// is not always ABI-safe to range over.
func NSArrayCount(array objc.ID) int {
	if array == 0 {
		return 0
	}
	return int(objc.Send[uint](array, selCount))
}

// NSArrayObjectAtIndex returns the element id at index i, or 0 for a nil array.
func NSArrayObjectAtIndex(array objc.ID, i int) objc.ID {
	if array == 0 {
		return 0
	}
	return objc.Send[objc.ID](array, selObjectAtIndex, uint(i))
}

// NSArrayToSlice converts an NSArray object to a Go slice, mapping each element
// id through conv. Returns nil for a nil or empty array. This is the reliable
// way to iterate an NSArray returned from the bindings (the generated typed
// NSArray[T] value cannot always be ranged over directly).
func NSArrayToSlice[T any](array objc.ID, conv func(objc.ID) T) []T {
	n := NSArrayCount(array)
	if n == 0 {
		return nil
	}
	out := make([]T, n)
	for i := range out {
		out[i] = conv(objc.Send[objc.ID](array, selObjectAtIndex, uint(i)))
	}
	return out
}

// SliceToNSArray builds an autoreleased NSMutableArray from a Go slice, mapping
// each element to an id through conv. Retain the result if it must outlive the
// current autorelease pool drain.
func SliceToNSArray[T any](items []T, conv func(T) objc.ID) objc.ID {
	arr := objc.Send[objc.ID](objc.ID(clsNSMutableArray), selArray)
	for _, item := range items {
		arr.Send(selAddObject, conv(item))
	}
	return arr
}
