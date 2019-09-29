package log

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Wrap every zap field function to remove zap from application imports whe using log pkg.

// Any type wrapper
func Any(key string, value interface{}) zapcore.Field {
	return zap.Any(key, value)
}

// Array type wrapper
func Array(key string, val zapcore.ArrayMarshaler) zapcore.Field {
	return zap.Array(key, val)
}

// Binary type wrapper
func Binary(key string, val []byte) zapcore.Field {
	return zap.Binary(key, val)
}

// Bool type wrapper
func Bool(key string, val bool) zapcore.Field {
	return zap.Bool(key, val)
}

// Bools type wrapper
func Bools(key string, bs []bool) zapcore.Field {
	return zap.Bools(key, bs)
}

// ByteString type wrapper
func ByteString(key string, val []byte) zapcore.Field {
	return zap.ByteString(key, val)
}

// ByteStrings type wrapper
func ByteStrings(key string, bss [][]byte) zapcore.Field {
	return zap.ByteStrings(key, bss)
}

// Complex128 type wrapper
func Complex128(key string, val complex128) zapcore.Field {
	return zap.Complex128(key, val)
}

// Complex128s type wrapper
func Complex128s(key string, nums []complex128) zapcore.Field {
	return zap.Complex128s(key, nums)
}

// Complex64 type wrapper
func Complex64(key string, val complex64) zapcore.Field {
	return zap.Complex64(key, val)
}

// Complex64s type wrapper
func Complex64s(key string, nums []complex64) zapcore.Field {
	return zap.Complex64s(key, nums)
}

// Duration type wrapper
func Duration(key string, val time.Duration) zapcore.Field {
	return zap.Duration(key, val)
}

// Durations type wrapper
func Durations(key string, ds []time.Duration) zapcore.Field {
	return zap.Durations(key, ds)
}

// Error type wrapper
func Error(err error) zapcore.Field {
	return zap.Error(err)
}

// Errors type wrapper
func Errors(key string, errs []error) zapcore.Field {
	return zap.Errors(key, errs)
}

// Float32 type wrapper
func Float32(key string, val float32) zapcore.Field {
	return zap.Float32(key, val)
}

// Float32s type wrapper
func Float32s(key string, nums []float32) zapcore.Field {
	return zap.Float32s(key, nums)
}

// Float64 type wrapper
func Float64(key string, val float64) zapcore.Field {
	return zap.Float64(key, val)
}

// Float64s type wrapper
func Float64s(key string, nums []float64) zapcore.Field {
	return zap.Float64s(key, nums)
}

// Int type wrapper
func Int(key string, val int) zapcore.Field {
	return zap.Int(key, val)
}

// Int16 type wrapper
func Int16(key string, val int16) zapcore.Field {
	return zap.Int16(key, val)
}

// Int16s type wrapper
func Int16s(key string, nums []int16) zapcore.Field {
	return zap.Int16s(key, nums)
}

// Int32 type wrapper
func Int32(key string, val int32) zapcore.Field {
	return zap.Int32(key, val)
}

// Int32s type wrapper
func Int32s(key string, nums []int32) zapcore.Field {
	return zap.Int32s(key, nums)
}

// Int64 type wrapper
func Int64(key string, val int64) zapcore.Field {
	return zap.Int64(key, val)
}

// Int64s type wrapper
func Int64s(key string, nums []int64) zapcore.Field {
	return zap.Int64s(key, nums)
}

// Int8 type wrapper
func Int8(key string, val int8) zapcore.Field {
	return zap.Int8(key, val)
}

// Int8s type wrapper
func Int8s(key string, nums []int8) zapcore.Field {
	return zap.Int8s(key, nums)
}

// Ints type wrapper
func Ints(key string, nums []int) zapcore.Field {
	return zap.Ints(key, nums)
}

// NamedError type wrapper
func NamedError(key string, err error) zapcore.Field {
	return zap.NamedError(key, err)
}

// Namespace type wrapper
func Namespace(key string) zapcore.Field {
	return zap.Namespace(key)
}

// Object type wrapper
func Object(key string, val zapcore.ObjectMarshaler) zapcore.Field {
	return zap.Object(key, val)
}

// Reflect type wrapper
func Reflect(key string, val interface{}) zapcore.Field {
	return zap.Reflect(key, val)
}

// Skip type wrapper
func Skip() zapcore.Field {
	return zap.Skip()
}

// Stack type wrapper
func Stack(key string) zapcore.Field {
	return zap.Stack(key)
}

// String type wrapper
func String(key string, val string) zapcore.Field {
	return zap.String(key, val)
}

// Stringer type wrapper
func Stringer(key string, val fmt.Stringer) zapcore.Field {
	return zap.Stringer(key, val)
}

// Strings type wrapper
func Strings(key string, ss []string) zapcore.Field {
	return zap.Strings(key, ss)
}

// Time type wrapper
func Time(key string, val time.Time) zapcore.Field {
	return zap.Time(key, val)
}

// Times type wrapper
func Times(key string, ts []time.Time) zapcore.Field {
	return zap.Times(key, ts)
}

// Uint type wrapper
func Uint(key string, val uint) zapcore.Field {
	return zap.Uint(key, val)
}

// Uint16 type wrapper
func Uint16(key string, val uint16) zapcore.Field {
	return zap.Uint16(key, val)
}

// Uint16s type wrapper
func Uint16s(key string, nums []uint16) zapcore.Field {
	return zap.Uint16s(key, nums)
}

// Uint32 type wrapper
func Uint32(key string, val uint32) zapcore.Field {
	return zap.Uint32(key, val)
}

// Uint32s type wrapper
func Uint32s(key string, nums []uint32) zapcore.Field {
	return zap.Uint32s(key, nums)
}

// Uint64 type wrapper
func Uint64(key string, val uint64) zapcore.Field {
	return zap.Uint64(key, val)
}

// Uint64s type wrapper
func Uint64s(key string, nums []uint64) zapcore.Field {
	return zap.Uint64s(key, nums)
}

// Uint8 type wrapper
func Uint8(key string, val uint8) zapcore.Field {
	return zap.Uint8(key, val)
}

// Uint8s type wrapper
func Uint8s(key string, nums []uint8) zapcore.Field {
	return zap.Uint8s(key, nums)
}

// Uintptr type wrapper
func Uintptr(key string, val uintptr) zapcore.Field {
	return zap.Uintptr(key, val)
}

// Uintptrs type wrapper
func Uintptrs(key string, us []uintptr) zapcore.Field {
	return zap.Uintptrs(key, us)
}

// Uints type wrapper
func Uints(key string, nums []uint) zapcore.Field {
	return zap.Uints(key, nums)
}
