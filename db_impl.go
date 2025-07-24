package sqlboiler

import (
	"encoding/json"
	"fmt"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/types"
	"github.com/elliotchance/pie/v2"
	jsonUtils "github.com/hdget/utils/json"
	"github.com/pkg/errors"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type DbImpl interface {
	Exclude(...string) DbImpl                         // 设置排除字段
	AutoIncr(...string) DbImpl                        // 设置自增字段
	JSONArray() DbImpl                                // 设置Json字段的处理函数为Json数组
	Fill(modelObject any, props map[string]any) error // 将props值填入到modelObject中
	Executor() boil.Executor
	Tid() int64
}

type dbImpl struct {
	tx             Transactor
	tid            int64
	excludeFields  []string
	autoIncrFields []string
	getJSONData    func(...any) []byte
}

// 预定义常用类型反射对象避免重复创建
var (
	timeType           = reflect.TypeOf(time.Time{})
	jsonType           = reflect.TypeOf(types.JSON{})
	bytesType          = reflect.TypeOf([]byte{})
	rawMessageType     = reflect.TypeOf(json.RawMessage{})
	errOverflow        = errors.New("integer overflow")
	errUnsupportedType = errors.New("unsupported field type for increment")
)

func Tdb(tid int64, tx ...Transactor) DbImpl {
	impl := &dbImpl{
		excludeFields:  []string{"id"},
		autoIncrFields: []string{"version"},
		getJSONData:    jsonUtils.JsonObject,
		tid:            tid,
	}
	if len(tx) > 0 {
		impl.tx = tx[0]
	}
	return impl
}

func Gdb(tx ...Transactor) DbImpl {
	impl := &dbImpl{
		excludeFields:  []string{"id"},
		autoIncrFields: []string{"version"},
		getJSONData:    jsonUtils.JsonObject,
	}
	if len(tx) > 0 {
		impl.tx = tx[0]
	}
	return impl
}

func (impl *dbImpl) Executor() boil.Executor {
	if impl.tx != nil {
		return impl.tx.Executor()
	}
	return boil.GetDB()
}

func (impl *dbImpl) Tid() int64 {
	return impl.tid
}

func (impl *dbImpl) Exclude(fields ...string) DbImpl {
	impl.excludeFields = fields
	return impl
}

func (impl *dbImpl) AutoIncr(fields ...string) DbImpl {
	impl.autoIncrFields = fields
	return impl
}

// JSONArray 设置Json字段为JSON Array类型
func (impl *dbImpl) JSONArray() DbImpl {
	impl.getJSONData = jsonUtils.JsonArray
	return impl
}

func (impl *dbImpl) Fill(object any, props map[string]any) error {
	// 检查目标结构体必须为指针
	objectVal := reflect.ValueOf(object)
	if objectVal.Kind() != reflect.Ptr || objectVal.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("object must be a point of struct")
	}

	objectVal = objectVal.Elem()
	for key, value := range props {
		// 排除不需要的字段
		if pie.Contains(impl.excludeFields, key) {
			continue
		}

		field := objectVal.FieldByName(capitalize(key))
		if !field.IsValid() || !field.CanSet() {
			continue // 忽略无效或不可导出字段
		}

		// 类型转换并设置字段值
		if pie.Contains(impl.autoIncrFields, strings.ToLower(key)) {
			if err := impl.incrFieldValue(field); err != nil {
				return errors.Wrap(err, "increase field value")
			}
		} else {
			if err := impl.setField(field, value); err != nil {
				return errors.Wrapf(err, "set field '%s'", field.Type().Name())
			}
		}
	}

	return nil
}

// 设置字段值（带类型转换）
func (impl *dbImpl) setField(field reflect.Value, value any) error {
	val := reflect.ValueOf(value)

	// 快速路径：类型完全匹配
	if value != nil && val.Type().AssignableTo(field.Type()) {
		field.Set(val)
		return nil
	}

	// 次快路径：类型可转换
	if value != nil && val.Type().ConvertibleTo(field.Type()) {
		field.Set(val.Convert(field.Type()))
		return nil
	}

	// 基础类型快速处理
	switch field.Kind() {
	case reflect.String:
		if s, ok := value.(string); ok {
			field.SetString(s)
			return nil
		}
	case reflect.Int64, reflect.Int: // 将高频的提前
		if v, ok := impl.tryParseIntOptimized(value); ok {
			field.SetInt(v)
		}
	case reflect.Float32, reflect.Float64:
		if num, err := strconv.ParseFloat(fmt.Sprint(value), 64); err == nil {
			field.SetFloat(num)
			return nil
		}
	case reflect.Bool:
		if b, err := strconv.ParseBool(fmt.Sprint(value)); err == nil {
			field.SetBool(b)
			return nil
		}
	case reflect.Int8, reflect.Int16, reflect.Int32:
		if v, ok := impl.tryParseIntOptimized(value); ok {
			field.SetInt(v)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v, ok := impl.tryParseUint(value); ok {
			field.SetUint(v)
		}
	}

	// 特殊类型匹配
	switch field.Type() {
	case timeType: // 处理时间类型
		return impl.handleTimeField(field, value)
	case jsonType:
		field.Set(reflect.ValueOf(types.JSON(impl.getJSONData(value))))
		return nil
	case rawMessageType:
		field.Set(reflect.ValueOf(json.RawMessage(impl.getJSONData(value))))
		return nil
	case bytesType:
		field.SetBytes(impl.getJSONData(value))
		return nil
	}

	return fmt.Errorf("unsupported type: %s", field.Kind())
}

// increaseFieldValue 将数字字段自增
func (impl *dbImpl) incrFieldValue(field reflect.Value) error {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(field.Int() + 1)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// 防止uint溢出
		if field.Uint() < ^uint64(0) {
			field.SetUint(field.Uint() + 1)
			return nil
		}
		return errOverflow
	default:
		return errUnsupportedType
	}
}

//// nil值处理逻辑
//func (impl *dbImpl) handleNilValue(field reflect.Value) error {
//	switch field.Kind() {
//	case reflect.Ptr, reflect.Interface, reflect.Map:
//		field.Set(reflect.Zero(field.Type()))
//		return nil
//	default: // 静默忽略非指针类型的nil
//		return nil
//	}
//}

func (impl *dbImpl) tryParseIntOptimized(value any) (int64, bool) {
	switch v := value.(type) {
	case float64, int, int64: // 覆盖80%高频类型, 注意: json unmarshal后的数字会是float64l类型
		return reflect.ValueOf(v).Int(), true
	case string:
		if len(v) > 0 && v[0] >= '0' && v[0] <= '9' {
			n, err := strconv.ParseInt(v, 10, 64)
			return n, err == nil
		}
	default:
		// 低频类型二次匹配
		if n, ok := impl.tryParseFast(value); ok {
			return n, true
		}
		return 0, false
	}
	return 0, false
}

func (impl *dbImpl) tryParseUint(value any) (uint64, bool) {
	switch v := value.(type) {
	case int, int8, int16, int32, int64:
		if n := reflect.ValueOf(v).Int(); n >= 0 {
			return uint64(n), true
		}
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(v).Uint(), true
	case float32:
		return uint64(v), true
	case string:
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			return n, true
		}
	}
	return 0, false
}

func (impl *dbImpl) tryParseFast(value any) (int64, bool) {
	switch v := value.(type) {
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		if v <= math.MaxInt64 {
			return int64(v), true
		}
	case float32:
		return int64(v), true
	case float64:
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			return int64(v), true
		}
	}
	return 0, false
}

// 时间类型处理优化
func (impl *dbImpl) handleTimeField(field reflect.Value, value any) error {
	switch v := value.(type) {
	case time.Time:
		field.Set(reflect.ValueOf(v))
	case int64:
		field.Set(reflect.ValueOf(time.Unix(v, 0)))
	case string:
		if t, err := time.Parse(time.DateTime, v); err == nil {
			field.Set(reflect.ValueOf(t))
		} else {
			return fmt.Errorf("invalid time format: %w", err)
		}
	default:
		return fmt.Errorf("unsupported time source: %T", value)
	}
	return nil
}
