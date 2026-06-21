package gs2vm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
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
	Weapons       []WeaponContext
	NPCID         uint32
	This          map[string]any
	ServerFlags   map[string]string
	ServerOptions map[string]string
	FileRoot      string
	Socket        *SocketContext
}

type Result struct {
	Output          []string
	ClientTriggers  []ClientTrigger
	PlayerFlags     []PlayerFlag
	ServerFlags     []ServerFlag
	PlayerMessages  []PlayerMessage
	PlayerWeapons   []PlayerWeapon
	PlayerWarps     []PlayerWarp
	NPCActions      []NPCAction
	SocketActions   []SocketAction
	SocketUpdates   []SocketContext
	ScheduledEvents []ScheduledEvent
	This            map[string]any
	Err             string
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

type WeaponContext struct {
	Name  string
	Image string
}

type PlayerFlag struct {
	Account string
	Name    string
	Value   string
}

type ServerFlag struct {
	Name    string
	Value   string
	Deleted bool
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

type NPCAction struct {
	ID        uint32
	ShapeType int
	Width     int
	Height    int
	TileTypes []string
	Chat      string
	WarpLevel string
	WarpX     float64
	WarpY     float64
}

type SocketAction struct {
	Action           string
	Name             string
	ID               string
	Port             int
	Data             string
	PackageDelimiter string
}

type ScheduledEvent struct {
	Event string
	Delay float64
}

type SocketContext struct {
	Name             string
	ID               string
	IPAddress        string
	Port             int
	PackageDelimiter string
	Data             string
	IsConnected      bool
}

type scriptPlayerObject struct {
	account        string
	client         *goja.Object
	clientr        *goja.Object
	initialClient  map[string]string
	initialClientr map[string]string
}

var spcPattern = regexp.MustCompile(`(?i)\s+SPC\s+`)
var tabPattern = regexp.MustCompile(`(?i)([\w\]\)"'])\s+TAB\s+([\w\[\("'])`)
var nlPattern = regexp.MustCompile(`(?i)([\w\]\)"'])\s+NL\s+([\w\[\("'])`)
var concatPattern = regexp.MustCompile(`\s+@\s+`)
var tempAssignPattern = regexp.MustCompile(`\btemp\.([A-Za-z_][A-Za-z0-9_]*)\s*=`)
var enumPattern = regexp.MustCompile(`(?is)\benum\s*\{([^{}]*)\}`)
var arrayAssignPattern = regexp.MustCompile(`=\s*\{([^{}\n;]*)\}`)
var arrayArgPattern = regexp.MustCompile(`([,(]\s*)\{([^{}\n;]*)\}`)
var newArrayPattern = regexp.MustCompile(`new\s*\[([^\]]*)\]`)
var inArrayPattern = regexp.MustCompile(`(?i)([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*|"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|\d+(?:\.\d+)?)\s+in\s+\{([^{}\n;]*)\}`)
var dynamicCallPattern = regexp.MustCompile(`\(\s*@\s*([^)]+?)\s*\)\s*\(([^()]*)\)`)
var forKeywordPattern = regexp.MustCompile(`(?i)\bfor\s*\(`)
var tempForPattern = regexp.MustCompile(`\bfor\s*\(\s*temp\.([A-Za-z_][A-Za-z0-9_]*)\s*=([^;]*);([^;]*);([^)]*)\)\s*\{`)
var forEachPattern = regexp.MustCompile(`\bfor\s*\(\s*(temp\.)?([A-Za-z_][A-Za-z0-9_]*)\s*(?::|\bin\b)\s*([^)]+)\)\s*\{`)
var dottedTempParamFunctionPattern = regexp.MustCompile(`\bfunction\s+([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*temp\.([A-Za-z_][A-Za-z0-9_]*)\s*\)\s*\{`)
var dottedFunctionPattern = regexp.MustCompile(`\bfunction\s+([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
var tempParamFunctionPattern = regexp.MustCompile(`\bfunction\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*temp\.[^)]*)\)\s*\{`)
var newTSocketPattern = regexp.MustCompile(`\bnew\s+TSocket\s*\(`)

func Run(config Config) Result {
	vm := goja.New()
	result := Result{}
	src := Translate(StripClientside(config.Script))
	players := make([]scriptPlayerObject, 0, len(config.Players)+1)
	drawings := make(map[int64]*goja.Object)
	thisObj := objectFromAnyMap(vm, config.This)

	vm.Set("name", config.ScriptName)
	vm.Set("params", append([]string(nil), config.Params...))
	vm.Set("allplayers", playerListObject(vm, &result, config.Players, &players))
	vm.Set("weapons", weaponListObject(vm, config.Weapons))
	vm.Set("temp", vm.NewObject())
	vm.Set("TAB", "\t")
	vm.Set("NL", "\n")
	vm.Set("NULL", goja.Null())
	vm.Set("screenwidth", 1024)
	vm.Set("screenheight", 1024)
	currentPlayer := playerContextFromMap(config.Player, config.PlayerFlags)
	currentPlayerObject := playerObject(vm, &result, currentPlayer, &players)
	vm.Set("player", currentPlayerObject)
	vm.Set("client", currentPlayerObject.Get("client"))
	vm.Set("clientr", currentPlayerObject.Get("clientr"))
	vm.Set("chat", "")
	serverFlags := flagValues(config.ServerFlags, "server.")
	serverrFlags := flagValues(config.ServerFlags, "serverr.")
	serverObj := flagObject(vm, serverFlags)
	serverrObj := flagObject(vm, serverrFlags)
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
	vm.Set("setshape", func(call goja.FunctionCall) goja.Value {
		if config.NPCID != 0 {
			result.NPCActions = append(result.NPCActions, NPCAction{ID: config.NPCID, ShapeType: int(valueInt(call.Argument(0))), Width: int(valueInt(call.Argument(1))), Height: int(valueInt(call.Argument(2)))})
		}
		return goja.Undefined()
	})
	vm.Set("setshape2", func(call goja.FunctionCall) goja.Value {
		if config.NPCID != 0 {
			result.NPCActions = append(result.NPCActions, NPCAction{ID: config.NPCID, ShapeType: 2, Width: int(valueInt(call.Argument(0))), Height: int(valueInt(call.Argument(1))), TileTypes: valueLines(call.Argument(2))})
		}
		return goja.Undefined()
	})
	vm.Set("warpto", func(call goja.FunctionCall) goja.Value {
		if config.NPCID != 0 {
			result.NPCActions = append(result.NPCActions, NPCAction{ID: config.NPCID, WarpLevel: valueString(call.Argument(0)), WarpX: valueFloat(call.Argument(1)), WarpY: valueFloat(call.Argument(2))})
		}
		return goja.Undefined()
	})
	vm.Set("server", serverObj)
	vm.Set("serverr", serverrObj)
	vm.Set("serveroptions", objectFromMap(vm, config.ServerOptions))
	installFileFunctions(vm, config.FileRoot)
	installScriptUtilityFunctions(vm, &result, thisObj)
	installSocketFunctions(vm, &result)
	if config.Socket != nil {
		installCurrentSocketFunctions(vm, &result, config.Socket)
	}
	vm.Set("__callDynamic", func(call goja.FunctionCall) goja.Value {
		name := strings.TrimSpace(valueString(call.Argument(0)))
		if name == "" {
			return goja.Undefined()
		}
		fn, ok := goja.AssertFunction(vm.Get(name))
		if !ok {
			return goja.Undefined()
		}
		args := make([]goja.Value, 0, len(call.Arguments)-1)
		for _, arg := range call.Arguments[1:] {
			args = append(args, arg)
		}
		value, err := fn(goja.Undefined(), args...)
		if err != nil {
			panic(err)
		}
		return value
	})
	vm.Set("__gs2In", func(call goja.FunctionCall) goja.Value {
		needle := valueString(call.Argument(0))
		for _, candidate := range valueLines(call.Argument(1)) {
			if valueStringLiteral(candidate) == needle {
				return vm.ToValue(true)
			}
		}
		return vm.ToValue(false)
	})
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
	args := make([]goja.Value, 0, len(config.Params)+1)
	for _, param := range config.Params {
		args = append(args, vm.ToValue(param))
	}
	var socketObj *goja.Object
	if config.Socket != nil && len(args) == 0 {
		socketObj = newSocketObject(vm, &result, config.Socket.Name, config.Socket.ID, config.Socket)
		args = append(args, socketObj)
	}
	if _, err := fn(thisObj, args...); err != nil {
		result.Err = err.Error()
	}
	if config.Socket != nil && socketObj != nil {
		result.SocketUpdates = append(result.SocketUpdates, SocketContext{Name: config.Socket.Name, ID: config.Socket.ID, IPAddress: valueString(socketObj.Get("ipaddress")), Port: int(valueInt(socketObj.Get("port"))), PackageDelimiter: valueString(socketObj.Get("packagedelimiter")), Data: valueString(socketObj.Get("data")), IsConnected: !goja.IsNull(socketObj.Get("isconnected")) && !goja.IsUndefined(socketObj.Get("isconnected")) && socketObj.Get("isconnected").ToBoolean()})
	}
	collectPlayerFlags(vm, &result, players)
	collectServerFlagObject(&result, "server.", serverObj, serverFlags)
	collectServerFlagObject(&result, "serverr.", serverrObj, serverrFlags)
	if config.NPCID != 0 {
		if chat := valueString(vm.Get("chat")); chat != "" {
			result.NPCActions = append(result.NPCActions, NPCAction{ID: config.NPCID, Chat: chat})
		}
	}
	result.This = exportObject(thisObj)
	return result
}

func installCurrentSocketFunctions(vm *goja.Runtime, result *Result, socket *SocketContext) {
	vm.Set("outdatalength", 0)
	vm.Set("isconnected", socket.IsConnected)
	vm.Set("send", func(call goja.FunctionCall) goja.Value {
		result.SocketActions = append(result.SocketActions, SocketAction{Action: "send", Name: socket.Name, ID: socket.ID, Data: valueString(call.Argument(0))})
		return goja.Undefined()
	})
	vm.Set("close", func(call goja.FunctionCall) goja.Value {
		result.SocketActions = append(result.SocketActions, SocketAction{Action: "close", Name: socket.Name, ID: socket.ID})
		return goja.Undefined()
	})
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
	script = regexp.MustCompile(`(?i)\bpublic\s+function\b`).ReplaceAllString(script, `function`)
	script = dottedTempParamFunctionPattern.ReplaceAllString(script, `function ${1}_${2}(${3}) { temp.${3} = ${3};`)
	script = dottedFunctionPattern.ReplaceAllString(script, `function ${1}_${2}(`)
	script = translateTempParams(script)
	script = newTSocketPattern.ReplaceAllString(script, `__newTSocket(`)
	script = translateEnums(script)
	script = translateInArrays(script)
	script = arrayAssignPattern.ReplaceAllString(script, `= [$1]`)
	script = arrayArgPattern.ReplaceAllString(script, `$1[$2]`)
	script = newArrayPattern.ReplaceAllString(script, `new Array($1)`)
	script = translateDynamicCalls(script)
	script = forKeywordPattern.ReplaceAllString(script, `for (`)
	script = translateForEachLoops(script)
	script = translateTempForLoops(script)
	script = strings.ReplaceAll(script, ".size()", ".length")
	script = spcPattern.ReplaceAllString(script, ` + " " + `)
	script = tabPattern.ReplaceAllString(script, `$1 + "\t" + $2`)
	script = nlPattern.ReplaceAllString(script, `$1 + "\n" + $2`)
	script = strings.ReplaceAll(script, "@=", "+=")
	script = concatPattern.ReplaceAllString(script, ` + `)
	return aliasTempAssignments(script)
}

func translateInArrays(script string) string {
	return inArrayPattern.ReplaceAllString(script, `__gs2In($1, [$2])`)
}

func translateTempParams(script string) string {
	return tempParamFunctionPattern.ReplaceAllStringFunc(script, func(fn string) string {
		match := tempParamFunctionPattern.FindStringSubmatch(fn)
		if len(match) != 3 {
			return fn
		}
		rawParams := strings.Split(match[2], ",")
		params := make([]string, 0, len(rawParams))
		assigns := make([]string, 0, len(rawParams))
		for _, raw := range rawParams {
			name := strings.TrimSpace(raw)
			if strings.HasPrefix(name, "temp.") {
				name = strings.TrimSpace(strings.TrimPrefix(name, "temp."))
				assigns = append(assigns, "temp."+name+" = "+name+";")
			}
			params = append(params, name)
		}
		return "function " + match[1] + "(" + strings.Join(params, ", ") + ") { " + strings.Join(assigns, "")
	})
}

func installScriptUtilityFunctions(vm *goja.Runtime, result *Result, thisObj *goja.Object) {
	schedule := func(call goja.FunctionCall) goja.Value {
		result.ScheduledEvents = append(result.ScheduledEvents, ScheduledEvent{Delay: valueFloat(call.Argument(0)), Event: valueString(call.Argument(1))})
		return goja.Undefined()
	}
	noOp := func(call goja.FunctionCall) goja.Value { return goja.Undefined() }
	vm.Set("int", func(call goja.FunctionCall) goja.Value { return vm.ToValue(int(valueFloat(call.Argument(0)))) })
	vm.Set("random", func(call goja.FunctionCall) goja.Value {
		min := valueFloat(call.Argument(0))
		max := valueFloat(call.Argument(1))
		if max <= min {
			return vm.ToValue(min)
		}
		return vm.ToValue(min + rand.Float64()*(max-min))
	})
	vm.Set("char", func(call goja.FunctionCall) goja.Value { return vm.ToValue(string(rune(valueInt(call.Argument(0))))) })
	vm.Set("strlen", func(call goja.FunctionCall) goja.Value { return vm.ToValue(len(valueString(call.Argument(0)))) })
	vm.Set("isObject", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		return vm.ToValue(arg != nil && !goja.IsUndefined(arg) && !goja.IsNull(arg) && arg.ToObject(vm) != nil)
	})
	vm.Set("loadclass", noOp)
	vm.Set("join", noOp)
	vm.Set("leave", noOp)
	vm.Set("openurl", noOp)
	vm.Set("Adventure_setAllowedPortsBind", noOp)
	vm.Set("sleep", noOp)
	vm.Set("scheduleevent", schedule)
	vm.Set("scheduleEvent", schedule)
	vm.Set("replacetext", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(strings.ReplaceAll(valueString(call.Argument(0)), valueString(call.Argument(1)), valueString(call.Argument(2))))
	})
	vm.Set("toJson", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(stringifyJSON(call.Argument(0)))
	})
	thisObj.Set("scheduleevent", schedule)
	thisObj.Set("scheduleEvent", schedule)
	thisObj.Set("join", noOp)
	thisObj.Set("leave", noOp)
}

func stringifyJSON(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "null"
	}
	data, err := json.Marshal(value.Export())
	if err != nil {
		return "null"
	}
	return string(data)
}

func installSocketFunctions(vm *goja.Runtime, result *Result) {
	vm.Set("__newTSocket", func(call goja.FunctionCall) goja.Value {
		return newSocketObject(vm, result, valueString(call.Argument(0)), "", nil)
	})
}

func newSocketObject(vm *goja.Runtime, result *Result, name, id string, context *SocketContext) *goja.Object {
	if context != nil {
		if name == "" {
			name = context.Name
		}
		if id == "" {
			id = context.ID
		}
	}
	obj := vm.NewObject()
	obj.Set("__tsocketName", name)
	obj.Set("__tsocketID", id)
	obj.Set("name", name)
	obj.Set("objecttype", "TSocket")
	obj.Set("address", "")
	obj.Set("error", "")
	obj.Set("ipaddress", "")
	obj.Set("isconnected", false)
	obj.Set("port", 0)
	obj.Set("parent", goja.Null())
	obj.Set("data", "")
	obj.Set("packagedelimiter", "")
	obj.Set("enablessl", false)
	if context != nil {
		obj.Set("ipaddress", context.IPAddress)
		obj.Set("isconnected", context.IsConnected)
		obj.Set("port", context.Port)
		obj.Set("data", context.Data)
		obj.Set("packagedelimiter", context.PackageDelimiter)
	}
	obj.Set("bind", func(call goja.FunctionCall) goja.Value {
		result.SocketActions = append(result.SocketActions, SocketAction{Action: "bind", Name: name, ID: id, Port: int(valueInt(call.Argument(0))), PackageDelimiter: valueString(obj.Get("packagedelimiter"))})
		return goja.Undefined()
	})
	obj.Set("close", func(call goja.FunctionCall) goja.Value {
		result.SocketActions = append(result.SocketActions, SocketAction{Action: "close", Name: name, ID: id})
		return goja.Undefined()
	})
	obj.Set("destroy", func(call goja.FunctionCall) goja.Value {
		result.SocketActions = append(result.SocketActions, SocketAction{Action: "close", Name: name, ID: id})
		return goja.Undefined()
	})
	obj.Set("send", func(call goja.FunctionCall) goja.Value {
		result.SocketActions = append(result.SocketActions, SocketAction{Action: "send", Name: name, ID: id, Data: valueString(call.Argument(0))})
		return goja.Undefined()
	})
	obj.Set("join", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	return obj
}

func translateDynamicCalls(script string) string {
	return dynamicCallPattern.ReplaceAllStringFunc(script, func(call string) string {
		match := dynamicCallPattern.FindStringSubmatch(call)
		if len(match) != 3 {
			return call
		}
		target := strings.TrimSpace(match[1])
		args := strings.TrimSpace(match[2])
		if args == "" {
			return "__callDynamic(" + target + ")"
		}
		return "__callDynamic(" + target + ", " + args + ")"
	})
}

func translateForEachLoops(script string) string {
	return forEachPattern.ReplaceAllStringFunc(script, func(loop string) string {
		match := forEachPattern.FindStringSubmatch(loop)
		if len(match) != 4 {
			return loop
		}
		name := match[2]
		source := strings.TrimSpace(match[3])
		if match[1] != "" {
			return "for (" + name + " of " + source + ") { temp." + name + " = " + name + ";"
		}
		return "for (" + name + " of " + source + ") {"
	})
}

func translateTempForLoops(script string) string {
	return tempForPattern.ReplaceAllStringFunc(script, func(loop string) string {
		match := tempForPattern.FindStringSubmatch(loop)
		if len(match) != 5 {
			return loop
		}
		name := match[1]
		init := strings.TrimSpace(match[2])
		condition := strings.ReplaceAll(strings.TrimSpace(match[3]), "temp."+name, name)
		post := strings.ReplaceAll(strings.TrimSpace(match[4]), "temp."+name, name)
		return "for (" + name + " =" + init + "; " + condition + "; " + post + ") { temp." + name + " = " + name + ";"
	})
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

func installFileFunctions(vm *goja.Runtime, root string) {
	vm.Set("loadstring", func(call goja.FunctionCall) goja.Value {
		text, err := loadVMString(root, valueString(call.Argument(0)))
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue(text)
	})
	vm.Set("loadlines", func(call goja.FunctionCall) goja.Value {
		lines, err := loadVMLines(root, valueString(call.Argument(0)))
		if err != nil {
			return vm.ToValue([]string{})
		}
		return vm.ToValue(lines)
	})
	vm.Set("savestring", func(call goja.FunctionCall) goja.Value {
		err := saveVMString(root, valueString(call.Argument(0)), valueString(call.Argument(1)), saveMode(call.Argument(2)))
		return vm.ToValue(err == nil)
	})
	vm.Set("savelines", func(call goja.FunctionCall) goja.Value {
		err := saveVMLines(root, valueString(call.Argument(0)), valueLines(call.Argument(1)), saveMode(call.Argument(2)))
		return vm.ToValue(err == nil)
	})
	if arrayCtor := vm.Get("Array"); arrayCtor != nil {
		proto := arrayCtor.ToObject(vm).Get("prototype").ToObject(vm)
		proto.Set("add", func(call goja.FunctionCall) goja.Value {
			obj := call.This.ToObject(vm)
			length := int(valueInt(obj.Get("length")))
			for _, arg := range call.Arguments {
				obj.Set(strconv.Itoa(length), arg)
				length++
			}
			obj.Set("length", length)
			return call.This
		})
		proto.Set("savelines", func(call goja.FunctionCall) goja.Value {
			err := saveVMLines(root, valueString(call.Argument(0)), valueLines(call.This), saveMode(call.Argument(1)))
			return vm.ToValue(err == nil)
		})
		proto.Set("loadlines", func(call goja.FunctionCall) goja.Value {
			lines, err := loadVMLines(root, valueString(call.Argument(0)))
			if err != nil {
				lines = []string{}
			}
			obj := call.This.ToObject(vm)
			obj.Set("length", 0)
			for i, line := range lines {
				obj.Set(strconv.Itoa(i), line)
			}
			return call.This
		})
	}
	if stringCtor := vm.Get("String"); stringCtor != nil {
		proto := stringCtor.ToObject(vm).Get("prototype").ToObject(vm)
		proto.Set("substring", func(call goja.FunctionCall) goja.Value {
			text := valueString(call.This)
			start := int(valueInt(call.Argument(0)))
			if start < 0 {
				start = 0
			}
			if start > len(text) {
				start = len(text)
			}
			if len(call.Arguments) < 2 || goja.IsUndefined(call.Argument(1)) {
				return vm.ToValue(text[start:])
			}
			length := int(valueInt(call.Argument(1)))
			if length < 0 {
				length = 0
			}
			end := start + length
			if end > len(text) {
				end = len(text)
			}
			return vm.ToValue(text[start:end])
		})
		proto.Set("tokenize", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(strings.Split(valueString(call.This), valueString(call.Argument(0))))
		})
		proto.Set("pos", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(strings.Index(valueString(call.This), valueString(call.Argument(0))))
		})
		proto.Set("starts", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(strings.HasPrefix(valueString(call.This), valueString(call.Argument(0))))
		})
		proto.Set("ends", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(strings.HasSuffix(valueString(call.This), valueString(call.Argument(0))))
		})
		proto.Set("trim", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(strings.TrimSpace(valueString(call.This)))
		})
		proto.Set("lower", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(strings.ToLower(valueString(call.This)))
		})
		proto.Set("upper", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(strings.ToUpper(valueString(call.This)))
		})
		proto.Set("savestring", func(call goja.FunctionCall) goja.Value {
			err := saveVMString(root, valueString(call.Argument(0)), valueString(call.This), saveMode(call.Argument(1)))
			return vm.ToValue(err == nil)
		})
		proto.Set("loadstring", func(call goja.FunctionCall) goja.Value {
			text, err := loadVMString(root, valueString(call.Argument(0)))
			if err != nil {
				return vm.ToValue("")
			}
			return vm.ToValue(text)
		})
	}
	vm.Set("findfiles", func(call goja.FunctionCall) goja.Value {
		files, err := findVMFiles(root, valueString(call.Argument(0)), call.Argument(1).ToBoolean())
		if err != nil {
			return vm.ToValue([]string{})
		}
		return vm.ToValue(files)
	})
}

func resolveVMFile(root, name string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("missing file root")
	}
	clean := filepath.Clean(strings.ReplaceAll(valueStringLiteral(name), "\\", "/"))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || strings.Contains(clean, string([]byte{0})) {
		return "", fmt.Errorf("invalid path")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	full := filepath.Join(rootAbs, clean)
	rel, err := filepath.Rel(rootAbs, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes root")
	}
	return full, nil
}

func loadVMString(root, name string) (string, error) {
	path, err := resolveVMFile(root, name)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func loadVMLines(root, name string) ([]string, error) {
	text, err := loadVMString(root, name)
	if err != nil {
		return nil, err
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return []string{}, nil
	}
	return strings.Split(text, "\n"), nil
}

func saveVMString(root, name, text string, appendMode bool) error {
	path, err := resolveVMFile(root, name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	flag := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	file, err := os.OpenFile(path, flag, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(text)
	return err
}

func saveVMLines(root, name string, lines []string, appendMode bool) error {
	text := strings.Join(lines, "\n")
	if len(lines) > 0 {
		text += "\n"
	}
	return saveVMString(root, name, text, appendMode)
}

func findVMFiles(root, pattern string, recursive bool) ([]string, error) {
	clean := strings.ReplaceAll(valueStringLiteral(pattern), "\\", "/")
	if recursive && !strings.Contains(clean, "**") {
		dir := filepath.ToSlash(filepath.Dir(clean))
		if dir == "." {
			clean = "**/" + filepath.Base(clean)
		} else {
			clean = strings.TrimRight(dir, "/") + "/**/" + filepath.Base(clean)
		}
	}
	base, err := resolveVMFile(root, clean)
	if err != nil {
		return nil, err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var matches []string
	if strings.Contains(clean, "**") {
		prefix, suffix, _ := strings.Cut(clean, "**")
		start := rootAbs
		if strings.Trim(prefix, `/\`) != "" {
			start, err = resolveVMFile(root, strings.TrimSuffix(prefix, "/"))
			if err != nil {
				return nil, err
			}
		}
		err = filepath.WalkDir(start, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(rootAbs, path)
			if err != nil {
				return nil
			}
			ok, _ := filepath.Match(strings.TrimLeft(suffix, `/\`), filepath.Base(rel))
			if ok {
				matches = append(matches, filepath.ToSlash(rel))
			}
			return nil
		})
		return matches, err
	}
	raw, err := filepath.Glob(base)
	if err != nil {
		return nil, err
	}
	for _, path := range raw {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		rel, err := filepath.Rel(rootAbs, path)
		if err == nil {
			matches = append(matches, filepath.ToSlash(rel))
		}
	}
	return matches, nil
}

func saveMode(value goja.Value) bool {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return false
	}
	switch typed := value.Export().(type) {
	case bool:
		return typed
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case float64:
		return typed != 0
	case string:
		return typed == "1" || strings.EqualFold(typed, "true") || strings.EqualFold(typed, "append")
	}
	return false
}

func valueLines(value goja.Value) []string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return []string{}
	}
	exported := value.Export()
	switch typed := exported.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		lines := make([]string, 0, len(typed))
		for _, line := range typed {
			lines = append(lines, fmt.Sprint(line))
		}
		return lines
	default:
		return []string{fmt.Sprint(typed)}
	}
}

func valueStringLiteral(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "\x00", ""))
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
	eventName = strings.ReplaceAll(eventName, ".", "_")
	names := []string{eventName}
	if !strings.HasPrefix(strings.ToLower(eventName), "on") && eventName != "" {
		names = append(names, "on"+strings.ToUpper(eventName[:1])+eventName[1:])
	}
	global := vm.GlobalObject()
	for _, name := range names {
		if fn, ok := goja.AssertFunction(vm.Get(name)); ok {
			return fn, true
		}
		for _, key := range global.Keys() {
			if strings.EqualFold(key, name) {
				if fn, ok := goja.AssertFunction(global.Get(key)); ok {
					return fn, true
				}
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

func collectServerFlagObject(result *Result, prefix string, obj *goja.Object, initial map[string]string) {
	if obj == nil {
		return
	}
	seen := make(map[string]bool)
	for _, key := range obj.Keys() {
		seen[key] = true
		value := valueString(obj.Get(key))
		if initial[key] != value {
			result.ServerFlags = append(result.ServerFlags, ServerFlag{Name: prefix + key, Value: value})
		}
	}
	for key := range initial {
		if !seen[key] {
			result.ServerFlags = append(result.ServerFlags, ServerFlag{Name: prefix + key, Deleted: true})
		}
	}
}

func playerMatches(player PlayerContext, target string) bool {
	return strings.EqualFold(player.Account, target) || strings.EqualFold(player.Nick, target) || strings.EqualFold(player.Nickname, target)
}

func playerListObject(vm *goja.Runtime, result *Result, players []PlayerContext, tracked *[]scriptPlayerObject) []goja.Value {
	out := make([]goja.Value, 0, len(players))
	for _, player := range players {
		out = append(out, playerObject(vm, result, player, tracked))
	}
	return out
}

func weaponListObject(vm *goja.Runtime, weapons []WeaponContext) []goja.Value {
	out := make([]goja.Value, 0, len(weapons))
	for _, weapon := range weapons {
		obj := vm.NewObject()
		obj.Set("name", weapon.Name)
		obj.Set("image", weapon.Image)
		out = append(out, obj)
	}
	return out
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
