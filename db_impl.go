package sqlboiler

import (
	"fmt"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/types"
	"github.com/elliotchance/pie/v2"
	jsonUtils "github.com/hdget/utils/json"
	"github.com/hdget/utils/text"
	"github.com/pkg/errors"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type DbImpl interface {
	Exclude(fieldNames ...string) DbImpl    // 设置排除字段
	AutoIncr(fieldNames ...string) DbImpl   // 设置自增字段
	JSONArray(fieldNames ...string) DbImpl  // 设置为Json数组的字段
	JSONObject(fieldNames ...string) DbImpl // 设置Json字段的处理函数为Json数组
	Copy(destObject any, source any) error  // 将source值填入到modelObject中
	Executor() boil.Executor
	Tid() int64
}

type dbImpl struct {
	tx               Transactor
	tid              int64
	excludeFields    []string
	autoIncrFields   []string
	jsonArrayFields  []string
	jsonObjectFields []string
}

// 预定义常用类型反射对象避免重复创建
var (
	timeType           = reflect.TypeOf(time.Time{})
	errOverflow        = errors.New("integer overflow")
	errUnsupportedType = errors.New("unsupported field type for increment")
	// CreateExcludes 创建操作默认忽略的字段
	CreateExcludes = []string{"id", "sn", "version", "created", "updated", "createdat", "updatedat", "r", "l"}
	// EditExcludes 编辑操作默认忽略的字段
	EditExcludes          = []string{"id", "sn", "created", "updated", "createdat", "updatedat", "r", "l"}
	defaultAutoIncrFields = []string{"version"}
)

func Tdb(tid int64, tx ...Transactor) DbImpl {
	impl := &dbImpl{
		autoIncrFields:   defaultAutoIncrFields,
		excludeFields:    make([]string, 0),
		jsonArrayFields:  make([]string, 0),
		jsonObjectFields: make([]string, 0),
		tid:              tid,
	}
	if len(tx) > 0 {
		impl.tx = tx[0]
	}
	return impl
}

func Gdb(tx ...Transactor) DbImpl {
	impl := &dbImpl{
		autoIncrFields:   defaultAutoIncrFields,
		excludeFields:    make([]string, 0),
		jsonArrayFields:  make([]string, 0),
		jsonObjectFields: make([]string, 0),
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
	impl.excludeFields = pie.Map(fields, func(v string) string {
		return strings.ToLower(v)
	})

	return impl
}

func (impl *dbImpl) AutoIncr(fields ...string) DbImpl {
	impl.autoIncrFields = pie.Map(fields, func(v string) string {
		return strings.ToLower(v)
	})
	return impl
}

// JSONArray 设置Json字段为JSON Array类型
func (impl *dbImpl) JSONArray(fields ...string) DbImpl {
	impl.jsonArrayFields = pie.Map(fields, func(v string) string {
		return strings.ToLower(v)
	})
	return impl
}

// JSONObject 设置Json字段为JSON Object类型
func (impl *dbImpl) JSONObject(fields ...string) DbImpl {
	impl.jsonObjectFields = pie.Map(fields, func(v string) string {
		return strings.ToLower(v)
	})
	return impl
}

func (impl *dbImpl) Copy(dest any, src any) error {
	to, isPtr := indirect(reflect.ValueOf(dest))
	toType, _ := indirectType(to.Type())
	if !isPtr || toType.Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a point of struct")
	}

	from, _ := indirect(reflect.ValueOf(src))
	fromType, _ := indirectType(from.Type())

	switch fromType.Kind() {
	case reflect.Struct:
		return impl.copyFromStruct(to, toType, from, fromType)
	case reflect.Map:
		return impl.copyFromMap(to, src)
	default:
		return fmt.Errorf("unsupported src type %v", fromType.Name())
	}
}

func (impl *dbImpl) copyFromMap(to reflect.Value, from any) error {
	props, ok := from.(map[string]any)
	if !ok {
		return errors.New("source is not map[string]any")
	}

	for key, value := range props {
		lowerCasedKey := strings.ToLower(key)

		// 排除不需要的字段
		if pie.Contains(impl.excludeFields, lowerCasedKey) {
			continue
		}

		destField := to.FieldByNameFunc(func(s string) bool {
			return strings.ToLower(s) == lowerCasedKey
		})
		if !destField.IsValid() || !destField.CanSet() {
			continue // 忽略无效或不可导出字段
		}

		// 类型转换并设置字段值
		if pie.Contains(impl.autoIncrFields, lowerCasedKey) {
			if err := impl.incrField(destField, value); err != nil {
				return errors.Wrap(err, "increase field value")
			}
		} else if pie.Contains(impl.jsonObjectFields, lowerCasedKey) {
			impl.handleJsonField(destField, value, jsonUtils.JsonObject)
			return nil
		} else if pie.Contains(impl.jsonArrayFields, lowerCasedKey) {
			impl.handleJsonField(destField, value, jsonUtils.JsonArray)
			return nil
		} else {
			if err := impl.setField(destField, reflect.ValueOf(value), value); err != nil {
				return errors.Wrapf(err, "set field '%s'", destField.Type().Name())
			}
		}
	}

	return nil
}

func (impl *dbImpl) copyFromStruct(to reflect.Value, toType reflect.Type, from reflect.Value, fromType reflect.Type) error {
	// 收集需要拷贝的字段
	srcFieldName2srcField := make(map[string]reflect.Value)
	for i := 0; i < from.NumField(); i++ {
		srcField := from.Field(i)
		srcFieldName := fromType.Field(i).Name

		lowerCasedSrcFieldName := strings.ToLower(srcFieldName)

		if pie.Contains(impl.excludeFields, lowerCasedSrcFieldName) || // 过滤指定的字段
			!text.IsCapitalized(srcFieldName) || // 过滤未导出的字段
			isComplexType(srcField.Type()) { // 过滤复杂类型
			continue
		}

		srcFieldName2srcField[lowerCasedSrcFieldName] = srcField
	}

	for i := 0; i < to.NumField(); i++ {
		destField := to.Field(i)
		destFieldName := toType.Field(i).Name

		if !destField.IsValid() || !destField.CanSet() {
			continue
		}

		lowerFieldName := strings.ToLower(destFieldName)
		if srcField, exists := srcFieldName2srcField[lowerFieldName]; exists {
			// 类型转换并设置字段值
			if pie.Contains(impl.autoIncrFields, lowerFieldName) {
				if err := impl.incrField(destField, srcField.Interface()); err != nil {
					return errors.Wrap(err, "increase field value")
				}
			} else if pie.Contains(impl.jsonObjectFields, lowerFieldName) {
				impl.handleJsonField(destField, srcField.Interface(), jsonUtils.JsonObject)
				return nil
			} else if pie.Contains(impl.jsonArrayFields, lowerFieldName) {
				impl.handleJsonField(destField, srcField.Interface(), jsonUtils.JsonArray)
				return nil
			} else {
				srcField, _ = indirect(srcField)
				if err := impl.setField(destField, srcField, srcField.Interface()); err != nil {
					return errors.Wrapf(err, "set field '%s'", srcField.Type().Name())
				}
			}

		}
	}

	return nil
}

func (impl *dbImpl) setField(destField reflect.Value, srcField reflect.Value, srcFieldValue any) error {
	// 快速路径：类型完全匹配
	if srcFieldValue != nil && srcField.Type().AssignableTo(destField.Type()) {
		destField.Set(srcField)
		return nil
	}

	// 次快路径：类型可转换
	if srcFieldValue != nil && srcField.Type().ConvertibleTo(destField.Type()) {
		destField.Set(srcField.Convert(destField.Type()))
		return nil
	}

	// 基础类型快速处理
	switch destField.Kind() {
	case reflect.String:
		if v, ok := srcFieldValue.(string); ok {
			destField.SetString(v)
			return nil
		}
	case reflect.Int64, reflect.Int: // 将高频的提前
		if v, ok := impl.tryParseInt64(srcFieldValue); ok {
			destField.SetInt(v)
		}
	case reflect.Float32, reflect.Float64:
		if num, err := strconv.ParseFloat(fmt.Sprint(srcFieldValue), 64); err == nil {
			destField.SetFloat(num)
			return nil
		}
	case reflect.Bool:
		if b, err := strconv.ParseBool(fmt.Sprint(srcFieldValue)); err == nil {
			destField.SetBool(b)
			return nil
		}
	case reflect.Int8, reflect.Int16, reflect.Int32:
		if v, ok := impl.tryParseInt64(srcFieldValue); ok {
			destField.SetInt(v)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v, ok := impl.tryParseUint64(srcFieldValue); ok {
			destField.SetUint(v)
		}
	}

	// 特殊类型匹配
	switch destField.Type() {
	case timeType: // 处理时间类型
		return impl.handleTimeField(destField, srcFieldValue)
	}

	return fmt.Errorf("unsupported type: %s", destField.Kind())
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

func (impl *dbImpl) tryParseInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			return int64(v), true
		}
	case int32, int, int64: // 覆盖80%高频类型, 注意: json unmarshal后的数字会是float64l类型
		return reflect.ValueOf(v).Int(), true
	case string:
		if len(v) > 0 && v[0] >= '0' && v[0] <= '9' {
			n, err := strconv.ParseInt(v, 10, 64)
			return n, err == nil
		}
	default:
		// 低频类型二次匹配
		if n, ok := impl.tryParseNumberFast(value); ok {
			return n, true
		}
		return 0, false
	}
	return 0, false
}

func (impl *dbImpl) tryParseUint64(value any) (uint64, bool) {
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

func (impl *dbImpl) tryParseNumberFast(value any) (int64, bool) {
	switch v := value.(type) {
	case int8:
		return int64(v), true
	case int16:
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

func indirect(reflectValue reflect.Value) (reflect.Value, bool) {
	for reflectValue.Kind() == reflect.Ptr {
		return reflectValue.Elem(), true
	}
	return reflectValue, false
}

func indirectType(reflectType reflect.Type) (reflect.Type, bool) {
	for reflectType.Kind() == reflect.Ptr || reflectType.Kind() == reflect.Slice {
		return reflectType.Elem(), true
	}
	return reflectType, false
}

// increaseFieldValue 将数字字段自增
func (impl *dbImpl) incrField(destField reflect.Value, srcFieldValue any) error {
	switch destField.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, ok := impl.tryParseInt64(srcFieldValue)
		if !ok {
			return errors.New("value is not int64")
		}
		destField.SetInt(val + 1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, ok := impl.tryParseUint64(srcFieldValue)
		if !ok {
			return errors.New("value is not uint64")
		}

		// 防止uint溢出
		if destField.Uint() > ^uint64(0) {
			return errOverflow
		}

		destField.SetUint(val + 1)
	default:
		return errUnsupportedType
	}

	return nil
}

// increaseFieldValue 将数字字段自增
func (impl *dbImpl) handleJsonField(destField reflect.Value, srcFieldValue any, fn func(...any) []byte) {
	if isByteSlice(destField) {
		destField.Set(reflect.ValueOf(types.JSON(fn(srcFieldValue))))
	}
}

func isComplexType(t reflect.Type) bool {
	switch t.Kind() {
	// 基础类型直接排除
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		return false

	// 特殊处理：排除 []byte
	case reflect.Slice:
		return t.Elem().Kind() != reflect.Uint8 // 若元素不是 uint8（byte），视为复杂类型

	// 明确归类为复杂类型
	case reflect.Struct, reflect.Map, reflect.Func, reflect.Chan, reflect.Interface:
		return true

	// 指针/数组：递归检查其指向或包含的类型
	case reflect.Ptr, reflect.Array:
		return isComplexType(t.Elem())

	// 其他类型（如 UnsafePointer）视为基础类型
	default:
		return false
	}
}

func isByteSlice(v reflect.Value) bool {
	/// 之前已经有检测/ 需先确保v是有效值
	//if !v.IsValid() {
	//	return false
	//}
	// 检查底层类型为切片，且元素类型为uint8（即[]byte）
	return v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8
}
