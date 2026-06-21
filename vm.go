package gs2vm

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dop251/goja"
)

type Config struct {
	ScriptName    string
	EventName     string
	Script        string
	Params        []string
	Player        map[string]string
	PlayerFlags   map[string]string
	Players       []PlayerContext
	This          map[string]any
	ServerFlags   map[string]string
	ServerOptions map[string]string
}

type Result struct {
	Output         []string
	ClientTriggers []ClientTrigger
	PlayerFlags    []PlayerFlag
	PlayerMessages []PlayerMessage
	PlayerWeapons  []PlayerWeapon
	PlayerWarps    []PlayerWarp
	This           map[string]any
	Err            string
}

type ClientTrigger struct {
	Kind string
	Name string
	Args []string
}

type PlayerContext struct {
	Account  string
	Nick     string
	Nickname string
	Level    string
	Flags    map[string]string
}

type PlayerFlag struct {
	Account string
	Name    string
	Value   string
}

type PlayerMessage struct {
	Account string
	Message string
}

type PlayerWeapon struct {
	Account string
	Name    string
	Add     bool
}

type PlayerWarp struct {
	Account string
	Level   string
	X       float64
	Y       float64
}

type scriptPlayerObject struct {
	account        string
	client         *goja.Object
	clientr        *goja.Object
	initialClient  map[string]string
	initialClientr map[string]string
}

var spcPattern = regexp.MustCompile(`(?i)\s+SPC\s+`)
var concatPattern = regexp.MustCompile(`\s+@\s+`)
var tempAssignPattern = regexp.MustCompile(`\btemp\.([A-Za-z_][A-Za-z0-9_]*)\s*=`)
var enumPattern = regexp.MustCompile(`(?is)\benum\s*\{([^{}]*)\}`)
var arrayAssignPattern = regexp.MustCompile(`=\s*\{([^{}\n;]*)\}`)
var newArrayPattern = regexp.MustCompile(`new\s*\[([^\]]*)\]`)

func Run(config Config) Result {
	vm := goja.New()
	result := Result{}
	src := Translate(StripClientside(config.Script))
	players := make([]scriptPlayerObject, 0, len(config.Players)+1)
	drawings := make(map[int64]*goja.Object)
	thisObj := objectFromAnyMap(vm, config.This)

	vm.Set("name", config.ScriptName)
	vm.Set("params", append([]string(nil), config.Params...))
	vm.Set("temp", vm.NewObject())
	vm.Set("TAB", "\t")
	vm.Set("NL", "\n")
	vm.Set("screenwidth", 1024)
	vm.Set("screenheight", 1024)
	currentPlayer := playerContextFromMap(config.Player, config.PlayerFlags)
	currentPlayerObject := playerObject(vm, &result, currentPlayer, &players)
	vm.Set("player", currentPlayerObject)
	vm.Set("client", currentPlayerObject.Get("client"))
	vm.Set("clientr", currentPlayerObject.Get("clientr"))
	vm.Set("setlevel", func(call goja.FunctionCall) goja.Value {
		addPlayerWarp(&result, currentPlayer.Account, valueString(call.Argument(0)), 0, 0)
		return goja.Undefined()
	})
	vm.Set("setlevel2", func(call goja.FunctionCall) goja.Value {
		addPlayerWarp(&result, currentPlayer.Account, valueString(call.Argument(0)), valueFloat(call.Argument(1)), valueFloat(call.Argument(2)))
		return goja.Undefined()
	})
	vm.Set("addweapon", func(call goja.FunctionCall) goja.Value {
		addPlayerWeapon(&result, currentPlayer.Account, valueString(call.Argument(0)), true)
		return goja.Undefined()
	})
	vm.Set("removeweapon", func(call goja.FunctionCall) goja.Value {
		addPlayerWeapon(&result, currentPlayer.Account, valueString(call.Argument(0)), false)
		return goja.Undefined()
	})
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
	vm.Set("base64encode", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(base64.StdEncoding.EncodeToString([]byte(valueString(call.Argument(0)))))
	})
	vm.Set("base64decode", func(call goja.FunctionCall) goja.Value {
		decoded, err := base64.StdEncoding.DecodeString(valueString(call.Argument(0)))
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue(string(decoded))
	})
	vm.Set("getimgwidth", func(call goja.FunctionCall) goja.Value {
		if strings.TrimSpace(valueString(call.Argument(0))) == "" {
			return vm.ToValue(0)
		}
		return vm.ToValue(1)
	})
	vm.Set("getimgheight", func(call goja.FunctionCall) goja.Value {
		if strings.TrimSpace(valueString(call.Argument(0))) == "" {
			return vm.ToValue(0)
		}
		return vm.ToValue(1)
	})
	vm.Set("showimg", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 4 {
			return vm.ToValue(0)
		}
		index := valueInt(call.Argument(0))
		obj := drawings[index]
		if obj == nil {
			obj = vm.NewObject()
			obj.Set("rotation", 0)
			drawings[index] = obj
		}
		obj.Set("index", index)
		obj.Set("image", valueString(call.Argument(1)))
		obj.Set("x", valueString(call.Argument(2)))
		obj.Set("y", valueString(call.Argument(3)))
		return vm.ToValue(0)
	})
	vm.Set("findimg", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return goja.Null()
		}
		if obj := drawings[valueInt(call.Argument(0))]; obj != nil {
			return obj
		}
		return goja.Null()
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
	vm.Set("findplayer", func(call goja.FunctionCall) goja.Value {
		target := strings.TrimSpace(valueString(call.Argument(0)))
		for _, candidate := range config.Players {
			if playerMatches(candidate, target) {
				return playerObject(vm, &result, candidate, &players)
			}
		}
		return goja.Null()
	})

	if _, err := vm.RunString(src); err != nil {
		result.Err = err.Error()
		return result
	}
	fn, ok := findFunction(vm, config.EventName)
	if !ok {
		return result
	}
	if _, err := fn(thisObj); err != nil {
		result.Err = err.Error()
	}
	collectPlayerFlags(vm, &result, players)
	result.This = exportObject(thisObj)
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
	script = translateEnums(script)
	script = arrayAssignPattern.ReplaceAllString(script, `= [$1]`)
	script = newArrayPattern.ReplaceAllString(script, `new Array($1)`)
	script = strings.ReplaceAll(script, ".size()", ".length")
	script = spcPattern.ReplaceAllString(script, ` + " " + `)
	script = strings.ReplaceAll(script, "@=", "+=")
	script = concatPattern.ReplaceAllString(script, ` + `)
	return aliasTempAssignments(script)
}

func translateEnums(script string) string {
	return enumPattern.ReplaceAllStringFunc(script, func(block string) string {
		match := enumPattern.FindStringSubmatch(block)
		if len(match) != 2 {
			return block
		}
		names := strings.Split(match[1], ",")
		var out strings.Builder
		index := 0
		for _, raw := range names {
			name := strings.TrimSpace(raw)
			if idx := strings.Index(name, "//"); idx >= 0 {
				name = strings.TrimSpace(name[:idx])
			}
			if name == "" {
				continue
			}
			if out.Len() > 0 {
				out.WriteByte('\n')
			}
			out.WriteString("var ")
			out.WriteString(name)
			out.WriteString(" = ")
			out.WriteString(strconv.Itoa(index))
			out.WriteByte(';')
			index++
		}
		return out.String()
	})
}

func aliasTempAssignments(script string) string {
	matches := tempAssignPattern.FindAllStringSubmatchIndex(script, -1)
	if len(matches) == 0 {
		return script
	}
	var out strings.Builder
	last := 0
	for _, match := range matches {
		if match[1] < len(script) && script[match[1]] == '=' {
			continue
		}
		name := script[match[2]:match[3]]
		out.WriteString(script[last:match[0]])
		out.WriteString("temp.")
		out.WriteString(name)
		out.WriteString(" = ")
		out.WriteString(name)
		out.WriteString(" =")
		last = match[1]
	}
	out.WriteString(script[last:])
	return out.String()
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

func objectFromAnyMap(vm *goja.Runtime, values map[string]any) *goja.Object {
	obj := vm.NewObject()
	for key, value := range values {
		obj.Set(key, value)
	}
	return obj
}

func exportObject(obj *goja.Object) map[string]any {
	out := make(map[string]any)
	if obj == nil {
		return out
	}
	for _, key := range obj.Keys() {
		out[key] = obj.Get(key).Export()
	}
	return out
}

func playerContextFromMap(values map[string]string, flags map[string]string) PlayerContext {
	return PlayerContext{Account: values["account"], Nick: firstNonEmpty(values["nick"], values["nickname"]), Nickname: firstNonEmpty(values["nickname"], values["nick"]), Level: values["level"], Flags: flags}
}

func playerObject(vm *goja.Runtime, result *Result, player PlayerContext, players *[]scriptPlayerObject) *goja.Object {
	obj := vm.NewObject()
	obj.Set("account", player.Account)
	obj.Set("nick", firstNonEmpty(player.Nick, player.Nickname))
	obj.Set("nickname", firstNonEmpty(player.Nickname, player.Nick))
	obj.Set("level", player.Level)
	clientFlags := flagValues(player.Flags, "client.")
	clientrFlags := flagValues(player.Flags, "clientr.")
	client := flagObject(vm, clientFlags)
	clientr := flagObject(vm, clientrFlags)
	obj.Set("client", client)
	obj.Set("clientr", clientr)
	send := func(call goja.FunctionCall) goja.Value {
		if player.Account != "" {
			result.PlayerMessages = append(result.PlayerMessages, PlayerMessage{Account: player.Account, Message: valueString(call.Argument(0))})
		}
		return goja.Undefined()
	}
	obj.Set("sendpm", send)
	obj.Set("sendplayer", send)
	obj.Set("setlevel", func(call goja.FunctionCall) goja.Value {
		addPlayerWarp(result, player.Account, valueString(call.Argument(0)), 0, 0)
		return goja.Undefined()
	})
	obj.Set("setlevel2", func(call goja.FunctionCall) goja.Value {
		addPlayerWarp(result, player.Account, valueString(call.Argument(0)), valueFloat(call.Argument(1)), valueFloat(call.Argument(2)))
		return goja.Undefined()
	})
	obj.Set("addweapon", func(call goja.FunctionCall) goja.Value {
		addPlayerWeapon(result, player.Account, valueString(call.Argument(0)), true)
		return goja.Undefined()
	})
	obj.Set("removeweapon", func(call goja.FunctionCall) goja.Value {
		addPlayerWeapon(result, player.Account, valueString(call.Argument(0)), false)
		return goja.Undefined()
	})
	*players = append(*players, scriptPlayerObject{account: player.Account, client: client, clientr: clientr, initialClient: clientFlags, initialClientr: clientrFlags})
	return obj
}

func addPlayerWeapon(result *Result, account, name string, add bool) {
	if result == nil || account == "" || strings.TrimSpace(name) == "" {
		return
	}
	result.PlayerWeapons = append(result.PlayerWeapons, PlayerWeapon{Account: account, Name: strings.TrimSpace(name), Add: add})
}

func addPlayerWarp(result *Result, account, level string, x, y float64) {
	if result == nil || account == "" {
		return
	}
	result.PlayerWarps = append(result.PlayerWarps, PlayerWarp{Account: account, Level: level, X: x, Y: y})
}

func flagValues(flags map[string]string, prefix string) map[string]string {
	values := make(map[string]string)
	for key, value := range flags {
		if name, ok := strings.CutPrefix(strings.ToLower(key), strings.ToLower(prefix)); ok {
			values[name] = value
		}
	}
	return values
}

func flagObject(vm *goja.Runtime, flags map[string]string) *goja.Object {
	obj := vm.NewObject()
	for key, value := range flags {
		obj.Set(key, mapValue(value))
	}
	return obj
}

func collectPlayerFlags(vm *goja.Runtime, result *Result, players []scriptPlayerObject) {
	for _, player := range players {
		collectFlagObject(vm, result, player.account, "client.", player.client, player.initialClient)
		collectFlagObject(vm, result, player.account, "clientr.", player.clientr, player.initialClientr)
	}
}

func collectFlagObject(vm *goja.Runtime, result *Result, account, prefix string, obj *goja.Object, initial map[string]string) {
	if account == "" || obj == nil {
		return
	}
	for _, key := range obj.Keys() {
		value := valueString(obj.Get(key))
		if initial[key] != value {
			result.PlayerFlags = append(result.PlayerFlags, PlayerFlag{Account: account, Name: prefix + key, Value: value})
		}
	}
}

func playerMatches(player PlayerContext, target string) bool {
	return strings.EqualFold(player.Account, target) || strings.EqualFold(player.Nick, target) || strings.EqualFold(player.Nickname, target)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
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

func valueInt(value goja.Value) int64 {
	switch typed := value.Export().(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	case string:
		parsed, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed
	default:
		return int64(value.ToInteger())
	}
}

func valueFloat(value goja.Value) float64 {
	switch typed := value.Export().(type) {
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case float64:
		return typed
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed
	default:
		return value.ToFloat()
	}
}
