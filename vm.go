package gs2vm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dop251/goja"
)

type Config struct {
	ScriptName    string
	EventName     string
	Script        string
	Params        []string
	Player        map[string]string
	ServerFlags   map[string]string
	ServerOptions map[string]string
}

type Result struct {
	Output         []string
	ClientTriggers []ClientTrigger
	Err            string
}

type ClientTrigger struct {
	Kind string
	Name string
	Args []string
}

var spcPattern = regexp.MustCompile(`(?i)\s+SPC\s+`)

func Run(config Config) Result {
	vm := goja.New()
	result := Result{}
	src := Translate(StripClientside(config.Script))

	vm.Set("name", config.ScriptName)
	vm.Set("params", append([]string(nil), config.Params...))
	vm.Set("temp", vm.NewObject())
	vm.Set("player", objectFromMap(vm, config.Player))
	vm.Set("server", objectFromPrefixedMap(vm, config.ServerFlags, "server."))
	vm.Set("serverr", objectFromPrefixedMap(vm, config.ServerFlags, "serverr."))
	vm.Set("serveroptions", objectFromMap(vm, config.ServerOptions))
	vm.Set("echo", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			parts = append(parts, valueString(arg))
		}
		result.Output = append(result.Output, strings.Join(parts, " "))
		return goja.Undefined()
	})
	vm.Set("triggerclient", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		trigger := ClientTrigger{Kind: valueString(call.Argument(0)), Name: valueString(call.Argument(1))}
		for _, arg := range call.Arguments[2:] {
			trigger.Args = append(trigger.Args, valueString(arg))
		}
		result.ClientTriggers = append(result.ClientTriggers, trigger)
		return goja.Undefined()
	})

	if _, err := vm.RunString(src); err != nil {
		result.Err = err.Error()
		return result
	}
	fn, ok := findFunction(vm, config.EventName)
	if !ok {
		return result
	}
	if _, err := fn(goja.Undefined()); err != nil {
		result.Err = err.Error()
	}
	return result
}

func StripClientside(script string) string {
	normalized := strings.ReplaceAll(script, "\r\n", "\n")
	lower := strings.ToLower(normalized)
	idx := strings.Index(lower, "//#clientside")
	if idx >= 0 {
		return strings.TrimSpace(normalized[:idx])
	}
	return strings.TrimSpace(normalized)
}

func Translate(script string) string {
	return spcPattern.ReplaceAllString(script, ` + " " + `)
}

func findFunction(vm *goja.Runtime, eventName string) (goja.Callable, bool) {
	if fn, ok := goja.AssertFunction(vm.Get(eventName)); ok {
		return fn, true
	}
	global := vm.GlobalObject()
	for _, key := range global.Keys() {
		if strings.EqualFold(key, eventName) {
			if fn, ok := goja.AssertFunction(global.Get(key)); ok {
				return fn, true
			}
		}
	}
	return nil, false
}

func objectFromMap(vm *goja.Runtime, values map[string]string) *goja.Object {
	obj := vm.NewObject()
	for key, value := range values {
		obj.Set(key, mapValue(value))
	}
	return obj
}

func objectFromPrefixedMap(vm *goja.Runtime, values map[string]string, prefix string) *goja.Object {
	obj := vm.NewObject()
	for key, value := range values {
		if name, ok := strings.CutPrefix(strings.ToLower(key), strings.ToLower(prefix)); ok {
			obj.Set(name, mapValue(value))
		}
	}
	return obj
}

func mapValue(value string) any {
	if strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}
	return value
}

func valueString(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return ""
	}
	exported := value.Export()
	switch typed := exported.(type) {
	case []string:
		return strings.Join(typed, ",")
	case []any:
		parts := make([]string, 0, len(typed))
		for _, part := range typed {
			parts = append(parts, fmt.Sprint(part))
		}
		return strings.Join(parts, ",")
	default:
		return value.String()
	}
}
