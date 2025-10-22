package sqlboiler

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/aarondl/sqlboiler/v4/types"
	"github.com/elliotchance/pie/v2"
	jsonUtils "github.com/hdget/utils/json"
	"github.com/hdget/utils/text"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
)

type DbCopier interface {
	Blacklist(fieldNames ...string) DbCopier                               // 设置黑名单字段
	Whitelist(fieldNames ...string) DbCopier                               // 设置白名单字段
	AutoIncr(fieldNames ...string) DbCopier                                // 设置自增字段
	JSONArray(fieldNames ...string) DbCopier                               // 设置为Json数组的字段
	JSONObject(fieldNames ...string) DbCopier                              // 设置Json字段的处理函数为Json数组
	Copy(destObject any, source any) error                                 // 将source值填入到modelObject中
	CopyForCreate(destObject any, source any, allowFields ...string) error // 创建动作需要的复制
	CopyForEdit(destObject any, source any, allowFields ...string) error   // 创建动作需要的复制
}

type dbCopierImpl struct {
	blacklistFields  map[string]struct{}
	whitelistFields  map[string]struct{}
	autoIncrFields   map[string]struct{}
	jsonArrayFields  map[string]struct{}
	jsonObjectFields map[string]struct{}
}

// 预定义常用类型反射对象避免重复创建
var (
	timeType           = reflect.TypeOf(time.Time{})
	errOverflow        = errors.New("integer overflow")
	errUnsupportedType = errors.New("unsupported field type for increment")

	// editSkipFields 编辑时默认忽略的字段
	createSkipFields = map[string]struct{}{
		"created_at": {},
		"updated_at": {},
		"version":    {},
		"tid":        {},
		"deleted_at": {},
		"r":          {},
		"l":          {},
	}

	// editSkipFields 编辑时默认忽略的字段
	editSkipFields = map[string]struct{}{
		"created_at": {},
		"updated_at": {},
		"version":    {},
		"tid":        {},
		"id":         {},
		"sn":         {},
		"deleted_at": {},
		"r":          {},
		"l":          {},
	}

	// 自增字段
	defaultAutoIncrFields = map[string]struct{}{
		"version": {},
	}
)

func newDbCopier() DbCopier {
	return &dbCopierImpl{
		autoIncrFields:   defaultAutoIncrFields,
		blacklistFields:  make(map[string]struct{}),
		whitelistFields:  make(map[string]struct{}),
		jsonArrayFields:  make(map[string]struct{}),
		jsonObjectFields: make(map[string]struct{}),
	}
}

// Blacklist 目标对象中除去blacklist的字段都会尝试拷贝
func (impl *dbCopierImpl) Blacklist(fields ...string) DbCopier {
	blacklistFields := pie.Map(fields, func(v string) string {
		return format(v)
	})

	for _, field := range blacklistFields {
		impl.blacklistFields[field] = struct{}{}
	}

	return impl
}

// Whitelist 目标对象中whitelist中出现的地段才会拷贝
func (impl *dbCopierImpl) Whitelist(fields ...string) DbCopier {
	whitelistFields := pie.Map(fields, func(v string) string {
		return format(v)
	})

	for _, field := range whitelistFields {
		impl.whitelistFields[field] = struct{}{}
	}

	return impl
}

func (impl *dbCopierImpl) AutoIncr(fields ...string) DbCopier {
	autoIncrFields := pie.Map(fields, func(v string) string {
		return format(v)
	})

	for _, field := range autoIncrFields {
		impl.autoIncrFields[field] = struct{}{}
	}
	return impl
}

// JSONArray 设置Json字段为JSON Array类型
func (impl *dbCopierImpl) JSONArray(fields ...string) DbCopier {
	jsonArrayFields := pie.Map(fields, func(v string) string {
		return format(v)
	})

	for _, field := range jsonArrayFields {
		impl.jsonArrayFields[field] = struct{}{}
	}

	return impl
}

// JSONObject 设置Json字段为JSON Object类型
func (impl *dbCopierImpl) JSONObject(fields ...string) DbCopier {
	jsonObjectFields := pie.Map(fields, func(v string) string {
		return format(v)
	})

	for _, field := range jsonObjectFields {
		impl.jsonObjectFields[field] = struct{}{}
	}

	return impl
}

func (impl *dbCopierImpl) CopyForCreate(dest any, src any, allowFields ...string) error {
	impl.blacklistFields = createSkipFields
	for _, field := range allowFields {
		delete(impl.blacklistFields, field)
	}
	return impl.Copy(dest, src)
}

func (impl *dbCopierImpl) CopyForEdit(dest any, src any, allowFields ...string) error {
	impl.blacklistFields = editSkipFields
	for _, field := range allowFields {
		delete(impl.blacklistFields, field)
	}
	return impl.Copy(dest, src)
}

func (impl *dbCopierImpl) Copy(dest any, src any) error {
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

// copyFromMap 不区分大小写
func (impl *dbCopierImpl) copyFromMap(to reflect.Value, from any) error {
	props, ok := from.(map[string]any)
	if !ok {
		return errors.New("source is not map[string]any")
	}

	for key, value := range props {
		formattedKey := format(key)

		if len(impl.whitelistFields) > 0 { // 如果有白名单，不在白名单中的字段都忽略, 优先级高
			if _, exist := impl.whitelistFields[formattedKey]; !exist {
				continue
			}
		} else { // 黑名单优先级低
			if _, exist := impl.blacklistFields[formattedKey]; exist {
				continue
			}
		}

		destField := to.FieldByNameFunc(func(field string) bool {
			return format(field) == formattedKey
		})
		if !destField.IsValid() || !destField.CanSet() {
			continue // 忽略无效或不可导出字段
		}

		// 类型转换并设置字段值
		if _, exist := impl.autoIncrFields[formattedKey]; exist {
			if err := impl.incrField(destField, value); err != nil {
				return errors.Wrap(err, "increase field value")
			}
		} else if _, exist := impl.jsonObjectFields[formattedKey]; exist {
			impl.handleJsonField(destField, value, jsonUtils.JsonObject)
		} else if _, exist := impl.jsonArrayFields[formattedKey]; exist {
			impl.handleJsonField(destField, value, jsonUtils.JsonArray)
		} else {
			if err := impl.setField(destField, reflect.ValueOf(value), value); err != nil {
				return errors.Wrapf(err, "set field '%s'", destField.Type().Name())
			}
		}
	}

	return nil
}

func (impl *dbCopierImpl) copyFromStruct(to reflect.Value, toType reflect.Type, from reflect.Value, fromType reflect.Type) error {
	// 收集需要拷贝的字段
	srcFieldName2srcField := make(map[string]reflect.Value)
	for i := 0; i < from.NumField(); i++ {
		srcField := from.Field(i)
		srcFieldName := fromType.Field(i).Name

		srcFormattedFieldName := format(srcFieldName)

		if len(impl.whitelistFields) > 0 { // 如果有白名单，不在白名单中的字段都忽略, 优先级高
			if _, exist := impl.whitelistFields[srcFormattedFieldName]; !exist {
				continue
			}
		} else { // 黑名单优先级低, 在黑名单中的都忽略
			if _, exist := impl.blacklistFields[srcFormattedFieldName]; exist {
				continue
			}
		}

		if !text.IsCapitalized(srcFieldName) || // 过滤未导出的字段
			!isSupportedType(srcField.Type()) { // 过滤不支持的类型
			continue
		}

		srcFieldName2srcField[srcFormattedFieldName] = srcField
	}

	for i := 0; i < to.NumField(); i++ {
		destField := to.Field(i)
		destFieldName := toType.Field(i).Name

		if !destField.IsValid() || !destField.CanSet() {
			continue
		}

		destFormattedFieldName := format(destFieldName)
		if srcField, exists := srcFieldName2srcField[destFormattedFieldName]; exists {
			// 类型转换并设置字段值
			if _, exist := impl.autoIncrFields[destFormattedFieldName]; exist {
				if err := impl.incrField(destField, srcField.Interface()); err != nil {
					return errors.Wrap(err, "increase struct field value")
				}
			} else if _, exist := impl.jsonObjectFields[destFormattedFieldName]; exist {
				impl.handleJsonField(destField, srcField.Interface(), jsonUtils.JsonObject)
			} else if _, exist := impl.jsonArrayFields[destFormattedFieldName]; exist {
				impl.handleJsonField(destField, srcField.Interface(), jsonUtils.JsonArray)
			} else {
				srcField, _ = indirect(srcField)
				var srcValue any
				if srcField.IsValid() {
					srcValue = srcField.Interface()
				}
				if err := impl.setField(destField, srcField, srcValue); err != nil {
					return errors.Wrapf(err, "copy to struct field '%s'", destFieldName)
				}
			}

		}
	}

	return nil
}

func (impl *dbCopierImpl) setField(destField reflect.Value, srcField reflect.Value, srcFieldValue any) error {
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
//func (impl *dbCopierImpl) handleNilValue(field reflect.Value) error {
//	switch field.Kind() {
//	case reflect.Ptr, reflect.Interface, reflect.Map:
//		field.Set(reflect.Zero(field.Type()))
//		return nil
//	default: // 静默忽略非指针类型的nil
//		return nil
//	}
//}

func (impl *dbCopierImpl) tryParseInt64(value any) (int64, bool) {
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

func (impl *dbCopierImpl) tryParseUint64(value any) (uint64, bool) {
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

func (impl *dbCopierImpl) tryParseNumberFast(value any) (int64, bool) {
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
func (impl *dbCopierImpl) handleTimeField(field reflect.Value, value any) error {
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
func (impl *dbCopierImpl) incrField(destField reflect.Value, srcFieldValue any) error {
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
func (impl *dbCopierImpl) handleJsonField(destField reflect.Value, srcFieldValue any, fn func(...any) []byte) {
	if isByteSlice(destField) {
		destField.Set(reflect.ValueOf(types.JSON(fn(srcFieldValue))))
	} else {
		destField.Set(reflect.ValueOf(types.JSON(fn())))
	}
}

func format(s string) string {
	return strings.ToLower(strcase.ToSnake(s))
}

func isSupportedType(t reflect.Type) bool {
	switch t.Kind() {
	// 基础类型直接排除
	case reflect.String, reflect.Float64, reflect.Int64, reflect.Int,
		reflect.Struct, reflect.Map, reflect.Slice,
		reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Bool, reflect.Float32:
		return true
	// 指针/数组：递归检查其指向或包含的类型
	case reflect.Ptr, reflect.Array:
		return isSupportedType(t.Elem())

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
