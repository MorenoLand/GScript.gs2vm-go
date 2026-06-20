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
	PlayerFlags   map[string]string
	Players       []PlayerContext
	ServerFlags   map[string]string
	ServerOptions map[string]string
}

type Result struct {
	Output         []string
	ClientTriggers []ClientTrigger
	PlayerFlags    []PlayerFlag
	PlayerMessages []PlayerMessage
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

type scriptPlayerObject struct {
	account        string
	client         *goja.Object
	clientr        *goja.Object
	initialClient  map[string]string
	initialClientr map[string]string
}

var spcPattern = regexp.MustCompile(`(?i)\s+SPC\s+`)

func Run(config Config) Result {
	vm := goja.New()
	result := Result{}
	src := Translate(StripClientside(config.Script))
	players := make([]scriptPlayerObject, 0, len(config.Players)+1)

	vm.Set("name", config.ScriptName)
	vm.Set("params", append([]string(nil), config.Params...))
	vm.Set("temp", vm.NewObject())
	currentPlayer := playerContextFromMap(config.Player, config.PlayerFlags)
	currentPlayerObject := playerObject(vm, &result, currentPlayer, &players)
	vm.Set("player", currentPlayerObject)
	vm.Set("client", currentPlayerObject.Get("client"))
	vm.Set("clientr", currentPlayerObject.Get("clientr"))
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
	if _, err := fn(goja.Undefined()); err != nil {
		result.Err = err.Error()
	}
	collectPlayerFlags(vm, &result, players)
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
	*players = append(*players, scriptPlayerObject{account: player.Account, client: client, clientr: clientr, initialClient: clientFlags, initialClientr: clientrFlags})
	return obj
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
