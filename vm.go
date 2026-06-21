package gs2vm

import (
	"crypto/des"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
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
	NPCs          []NPCContext
	NPCID         uint32
	This          map[string]any
	ServerFlags   map[string]string
	ServerOptions map[string]string
	FileRoot      string
	FileRights    []string
	Socket        *SocketContext
}

type Result struct {
	Output           []string
	ClientTriggers   []ClientTrigger
	PlayerFlags      []PlayerFlag
	ServerFlags      []ServerFlag
	PlayerMessages   []PlayerMessage
	PlayerRCMessages []PlayerMessage
	RCMessages       []string
	NCMessages       []string
	PlayerWeapons    []PlayerWeapon
	PlayerWarps      []PlayerWarp
	FileActions      []FileAction
	NPCFlags         []NPCFlag
	NPCFunctionCalls []NPCFunctionCall
	NPCActions       []NPCAction
	SocketActions    []SocketAction
	SocketUpdates    []SocketContext
	ScheduledEvents  []ScheduledEvent
	This             map[string]any
	Err              string
}

type ClientTrigger struct {
	Kind string
	Name string
	Args []string
}

type PlayerContext struct {
	ID       uint16
	Account  string
	Nick     string
	Nickname string
	Level    string
	Flags    map[string]string
	Rights   []string
	Folders  []string
}

type WeaponContext struct {
	Name  string
	Image string
}

type NPCContext struct {
	ID     uint32
	Name   string
	Script string
	This   map[string]any
}

type PlayerFlag struct {
	Account string
	Name    string
	Value   string
}

type NPCFlag struct {
	ID    uint32
	Name  string
	Value string
}

type NPCFunctionCall struct {
	ID       uint32
	Name     string
	Function string
	Args     []string
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

type FileAction struct {
	Action string
	Name   string
	Data   string
	OK     bool
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

type scriptNPCObject struct {
	id      uint32
	name    string
	obj     *goja.Object
	initial map[string]any
}

var spcPattern = regexp.MustCompile(`(?i)\s+SPC\s+`)
var tabPattern = regexp.MustCompile(`(?i)([\w\]\)"'])\s+TAB\s+([\w\[\("'])`)
var nlPattern = regexp.MustCompile(`(?i)([\w\]\)"'])\s+NL\s+([\w\[\("'])`)
var concatPattern = regexp.MustCompile(`\s+@\s+`)
var dynamicPropertyPattern = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\s*\.\s*\(([^()\n;]+)\)`)
var stringNPCPropertyPattern = regexp.MustCompile(`(^|[^A-Za-z0-9_])\(\s*"([^"]+)"\s*\)\s*\.([A-Za-z_][A-Za-z0-9_]*)`)
var tempLoadStringPattern = regexp.MustCompile(`\btemp\.([A-Za-z_][A-Za-z0-9_]*)\.loadstring\s*\(([^)]*)\)`)
var tempAssignPattern = regexp.MustCompile(`\btemp\.([A-Za-z_][A-Za-z0-9_]*)\s*=`)
var oneLineFunctionPattern = regexp.MustCompile(`(?m)\bfunction\s+([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\s*\(([^)]*)\)\s*([^{}\n;][^\n;]*);`)
var enumPattern = regexp.MustCompile(`(?is)\benum\s*\{([^{}]*)\}`)
var arrayAssignPattern = regexp.MustCompile(`=\s*\{([^{}\n;]*)\}`)
var arrayArgPattern = regexp.MustCompile(`([,(]\s*)\{([^{}\n;]*)\}`)
var newArrayChainPattern = regexp.MustCompile(`new\s*((?:\[[^\]]*\])+)+`)
var newArrayDimensionPattern = regexp.MustCompile(`\[([^\]]*)\]`)
var inArrayPattern = regexp.MustCompile(`(?i)([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*|"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|\d+(?:\.\d+)?)\s+in\s+\{([^{}\n;]*)\}`)
var dynamicCallPattern = regexp.MustCompile(`\(\s*@\s*([^)]+?)\s*\)\s*\(([^()]*)\)`)
var forKeywordPattern = regexp.MustCompile(`(?i)\bfor\s*\(`)
var loopOpenPattern = regexp.MustCompile(`(?i)\b(for|while)\s*\([^{}]*\)\s*\{`)
var tempForPattern = regexp.MustCompile(`\bfor\s*\(\s*temp\.([A-Za-z_][A-Za-z0-9_]*)\s*=([^;]*);([^;]*);([^)]*)\)\s*\{`)
var forEachPattern = regexp.MustCompile(`\bfor\s*\(\s*(temp\.)?([A-Za-z_][A-Za-z0-9_]*)\s*(?::|\bin\b)\s*([^)]+)\)\s*\{`)
var oneLineForEachPattern = regexp.MustCompile(`\bfor\s*\(\s*(temp\.)?([A-Za-z_][A-Za-z0-9_]*)\s*(?::|\bin\b)\s*([^)]+)\)\s+([^{}\n;]+;?)`)
var dottedTempParamFunctionPattern = regexp.MustCompile(`\bfunction\s+([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*temp\.([A-Za-z_][A-Za-z0-9_]*)\s*\)\s*\{`)
var dottedFunctionPattern = regexp.MustCompile(`\bfunction\s+([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
var tempParamFunctionPattern = regexp.MustCompile(`\bfunction\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*temp\.[^)]*)\)\s*\{`)
var newTSocketPattern = regexp.MustCompile(`\bnew\s+TSocket\s*\(`)

func Run(config Config) Result {
	vm := goja.New()
	result := Result{}
	src := Translate(StripClientside(config.Script))
	players := make([]scriptPlayerObject, 0, len(config.Players)+1)
	npcs := make([]scriptNPCObject, 0, len(config.NPCs))
	drawings := make(map[int64]*goja.Object)
	thisObj := objectFromAnyMap(vm, config.This)
	if valueString(thisObj.Get("name")) == "" {
		thisObj.Set("name", config.ScriptName)
	}
	thisObj.Set("toString", func(call goja.FunctionCall) goja.Value { return vm.ToValue(config.ScriptName) })

	vm.Set("name", config.ScriptName)
	vm.Set("params", append([]string(nil), config.Params...))
	allPlayers := playerListObject(vm, &result, config.Players, &players)
	vm.Set("allplayers", allPlayers)
	vm.Set("players", allPlayers)
	vm.Set("weapons", weaponListObject(vm, config.Weapons))
	vm.Set("temp", vm.NewObject())
	vm.Set("maxlooplimit", 10000)
	vm.Set("TAB", "\t")
	vm.Set("NL", "\n")
	vm.Set("NULL", goja.Null())
	vm.Set("nil", goja.Null())
	vm.Set("screenwidth", 1024)
	vm.Set("screenheight", 1024)
	currentPlayer := playerContextFromMap(config.Player, config.PlayerFlags)
	currentPlayerObject := playerObject(vm, &result, currentPlayer, &players)
	installNPCObjects(vm, &result, config.NPCs, &npcs)
	vm.Set("player", currentPlayerObject)
	vm.Set("client", currentPlayerObject.Get("client"))
	vm.Set("clientr", currentPlayerObject.Get("clientr"))
	if isPlayerLifecycleEvent(config.EventName) && currentPlayer.Account != "" {
		vm.Set("params", vm.NewArray(currentPlayerObject))
	}
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
	vm.Set("sendpm", func(call goja.FunctionCall) goja.Value {
		account := valueString(call.Argument(0))
		message := valueString(call.Argument(1))
		if account != "" && message != "" {
			result.PlayerMessages = append(result.PlayerMessages, PlayerMessage{Account: account, Message: message})
		}
		return goja.Undefined()
	})
	vm.Set("sendplayer", func(call goja.FunctionCall) goja.Value {
		account := valueString(call.Argument(0))
		message := valueString(call.Argument(1))
		if account != "" && message != "" {
			result.PlayerMessages = append(result.PlayerMessages, PlayerMessage{Account: account, Message: message})
		}
		return goja.Undefined()
	})
	vm.Set("sendtorc", func(call goja.FunctionCall) goja.Value {
		if message := valueString(call.Argument(0)); message != "" {
			result.RCMessages = append(result.RCMessages, message)
		}
		return goja.Undefined()
	})
	vm.Set("sendtonc", func(call goja.FunctionCall) goja.Value {
		if message := valueString(call.Argument(0)); message != "" {
			result.NCMessages = append(result.NCMessages, message)
		}
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
	installFileFunctions(vm, config.FileRoot, config.FileRights)
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
	loopCount := 0
	vm.Set("__gs2LoopTick", func(call goja.FunctionCall) goja.Value {
		loopCount++
		limit := int(valueInt(vm.Get("maxlooplimit")))
		if limit <= 0 {
			limit = 10000
		}
		if loopCount > limit {
			panic(vm.NewTypeError("maxlooplimit exceeded"))
		}
		return goja.Undefined()
	})
	vm.Set("echo", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			parts = append(parts, valueString(arg))
		}
		result.Output = append(result.Output, strings.Join(parts, " "))
		return goja.Undefined()
	})
	vm.Set("trace", func(call goja.FunctionCall) goja.Value {
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
	vm.Set("des_encrypt", func(call goja.FunctionCall) goja.Value {
		data, err := desEncrypt(valueString(call.Argument(0)), valueString(call.Argument(1)))
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue(string(data))
	})
	vm.Set("des_decrypt", func(call goja.FunctionCall) goja.Value {
		data, err := desDecrypt(valueString(call.Argument(0)), []byte(valueString(call.Argument(1))))
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue(string(data))
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
	vm.Set("findnpc", func(call goja.FunctionCall) goja.Value {
		target := strings.TrimSpace(valueString(call.Argument(0)))
		for _, npc := range npcs {
			if strings.EqualFold(npc.name, target) {
				return npc.obj
			}
		}
		return goja.Null()
	})
	vm.Set("findnpcbyid", func(call goja.FunctionCall) goja.Value {
		target := uint32(valueInt(call.Argument(0)))
		for _, npc := range npcs {
			if npc.id == target {
				return npc.obj
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
	if isPlayerLifecycleEvent(config.EventName) && currentPlayer.Account != "" {
		args = append(args, currentPlayerObject)
	} else {
		for _, param := range config.Params {
			args = append(args, vm.ToValue(param))
		}
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
	collectNPCFlags(&result, npcs)
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
	script = translateOneLineFunctions(script)
	script = dottedTempParamFunctionPattern.ReplaceAllString(script, `function ${1}_${2}(${3}) { temp.${3} = ${3};`)
	script = dottedFunctionPattern.ReplaceAllString(script, `function ${1}_${2}(`)
	script = translateTempParams(script)
	script = newTSocketPattern.ReplaceAllString(script, `__newTSocket(`)
	script = translateEnums(script)
	script = translateInArrays(script)
	script = arrayAssignPattern.ReplaceAllString(script, `= [$1]`)
	script = arrayArgPattern.ReplaceAllString(script, `$1[$2]`)
	script = translateNewArrays(script)
	script = translateDynamicCalls(script)
	script = translateTempLoadString(script)
	script = forKeywordPattern.ReplaceAllString(script, `for (`)
	script = translateOneLineForEachLoops(script)
	script = translateForEachLoops(script)
	script = translateTempForLoops(script)
	script = injectLoopLimitTicks(script)
	script = strings.ReplaceAll(script, ".size()", ".length")
	script = translateDynamicProperties(script)
	script = spcPattern.ReplaceAllString(script, ` + " " + `)
	script = tabPattern.ReplaceAllString(script, `$1 + "\t" + $2`)
	script = nlPattern.ReplaceAllString(script, `$1 + "\n" + $2`)
	script = strings.ReplaceAll(script, "@=", "+=")
	script = concatPattern.ReplaceAllString(script, ` + `)
	return aliasTempAssignments(script)
}

func isPlayerLifecycleEvent(eventName string) bool {
	return strings.EqualFold(eventName, "onPlayerLogin") || strings.EqualFold(eventName, "onPlayerLogout")
}

func translateTempLoadString(script string) string {
	return tempLoadStringPattern.ReplaceAllString(script, `__gs2LoadStringVar(temp, "$1", $2)`)
}

func translateOneLineFunctions(script string) string {
	return oneLineFunctionPattern.ReplaceAllStringFunc(script, func(fn string) string {
		match := oneLineFunctionPattern.FindStringSubmatch(fn)
		if len(match) != 4 {
			return fn
		}
		return "function " + match[1] + "(" + match[2] + ") { " + strings.Join(splitTopLevelCommas(match[3]), "; ") + "; }"
	})
}

func splitTopLevelCommas(text string) []string {
	parts := []string{}
	start, depth := 0, 0
	quote := rune(0)
	escaped := false
	for i, r := range text {
		if quote != 0 {
			if escaped {
				escaped = false
			} else if r == '\\' {
				escaped = true
			} else if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '"', '\'':
			quote = r
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(text[start:i]))
				start = i + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(text[start:]))
	out := parts[:0]
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func translateDynamicProperties(script string) string {
	script = stringNPCPropertyPattern.ReplaceAllString(script, `${1}findnpc("$2").${3}`)
	return dynamicPropertyPattern.ReplaceAllString(script, `$1[$2]`)
}

func injectLoopLimitTicks(script string) string {
	return loopOpenPattern.ReplaceAllStringFunc(script, func(loop string) string {
		return loop + "__gs2LoopTick();"
	})
}

func translateInArrays(script string) string {
	return inArrayPattern.ReplaceAllString(script, `__gs2In($1, [$2])`)
}

func translateNewArrays(script string) string {
	return newArrayChainPattern.ReplaceAllStringFunc(script, func(expr string) string {
		matches := newArrayDimensionPattern.FindAllStringSubmatch(expr, -1)
		if len(matches) == 0 {
			return expr
		}
		dims := make([]string, 0, len(matches))
		for _, match := range matches {
			if len(match) > 1 {
				dims = append(dims, strings.TrimSpace(match[1]))
			}
		}
		if len(dims) == 1 {
			return "new Array(" + dims[0] + ")"
		}
		return "__newGS2Array(" + strings.Join(dims, ", ") + ")"
	})
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
	vm.Set("float", func(call goja.FunctionCall) goja.Value { return vm.ToValue(valueFloat(call.Argument(0))) })
	vm.Set("double", func(call goja.FunctionCall) goja.Value { return vm.ToValue(valueFloat(call.Argument(0))) })
	vm.Set("strtofloat", func(call goja.FunctionCall) goja.Value { return vm.ToValue(valueFloat(call.Argument(0))) })
	vm.Set("abs", func(call goja.FunctionCall) goja.Value { return vm.ToValue(math.Abs(valueFloat(call.Argument(0)))) })
	vm.Set("ceil", func(call goja.FunctionCall) goja.Value { return vm.ToValue(math.Ceil(valueFloat(call.Argument(0)))) })
	vm.Set("floor", func(call goja.FunctionCall) goja.Value { return vm.ToValue(math.Floor(valueFloat(call.Argument(0)))) })
	vm.Set("sin", func(call goja.FunctionCall) goja.Value { return vm.ToValue(math.Sin(valueFloat(call.Argument(0)))) })
	vm.Set("cos", func(call goja.FunctionCall) goja.Value { return vm.ToValue(math.Cos(valueFloat(call.Argument(0)))) })
	vm.Set("tan", func(call goja.FunctionCall) goja.Value { return vm.ToValue(math.Tan(valueFloat(call.Argument(0)))) })
	vm.Set("strequals", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(valueString(call.Argument(0)) == valueString(call.Argument(1)))
	})
	vm.Set("strcontains", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(strings.Contains(valueString(call.Argument(0)), valueString(call.Argument(1))))
	})
	vm.Set("contains", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(strings.Contains(valueString(call.Argument(0)), valueString(call.Argument(1))))
	})
	vm.Set("startswith", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(strings.HasPrefix(valueString(call.Argument(0)), valueString(call.Argument(1))))
	})
	vm.Set("endswith", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(strings.HasSuffix(valueString(call.Argument(0)), valueString(call.Argument(1))))
	})
	vm.Set("uppercase", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(strings.ToUpper(valueString(call.Argument(0))))
	})
	vm.Set("lowercase", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(strings.ToLower(valueString(call.Argument(0))))
	})
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
	vm.Set("format", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return vm.ToValue("")
		}
		format := valueString(call.Argument(0))
		args := make([]any, 0, len(call.Arguments)-1)
		for _, arg := range call.Arguments[1:] {
			args = append(args, arg.Export())
		}
		return vm.ToValue(fmt.Sprintf(format, args...))
	})
	vm.Set("getextension", func(call goja.FunctionCall) goja.Value {
		ext := strings.TrimPrefix(path.Ext(valueString(call.Argument(0))), ".")
		return vm.ToValue(ext)
	})
	vm.Set("hideimgs", func(call goja.FunctionCall) goja.Value { return vm.ToValue(0) })
	vm.Set("keycode", func(call goja.FunctionCall) goja.Value { return vm.ToValue(valueInt(call.Argument(0))) })
	vm.Set("__newGS2Array", func(call goja.FunctionCall) goja.Value {
		dims := make([]int, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			size := int(valueInt(arg))
			if size < 0 {
				size = 0
			}
			dims = append(dims, size)
		}
		return vm.ToValue(newGS2Array(dims))
	})
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
		item := "__gs2Item_" + name
		if match[1] != "" {
			return "for (let " + item + " of " + source + ") { temp." + name + " = " + item + "; " + name + " = " + item + ";"
		}
		return "for (let " + item + " of " + source + ") { " + name + " = " + item + ";"
	})
}

func translateOneLineForEachLoops(script string) string {
	return oneLineForEachPattern.ReplaceAllStringFunc(script, func(loop string) string {
		match := oneLineForEachPattern.FindStringSubmatch(loop)
		if len(match) != 5 {
			return loop
		}
		body := strings.TrimSpace(match[4])
		if !strings.HasSuffix(body, ";") {
			body += ";"
		}
		return "for (" + match[1] + match[2] + ": " + strings.TrimSpace(match[3]) + ") { " + body + " }"
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

func installFileFunctions(vm *goja.Runtime, root string, rights []string) {
	vm.Set("loadstring", func(call goja.FunctionCall) goja.Value {
		name := valueString(call.Argument(0))
		if !vmFileHasRight(rights, name, 'r') {
			return vm.ToValue("")
		}
		text, err := loadVMString(root, name)
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue(text)
	})
	vm.Set("__gs2LoadStringVar", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			return vm.ToValue(false)
		}
		obj := call.Argument(0).ToObject(vm)
		name := valueString(call.Argument(1))
		file := valueString(call.Argument(2))
		if obj == nil || name == "" || !vmFileHasRight(rights, file, 'r') {
			return vm.ToValue(false)
		}
		text, err := loadVMString(root, file)
		if err != nil {
			return vm.ToValue(false)
		}
		obj.Set(name, text)
		vm.Set(name, text)
		return vm.ToValue(true)
	})
	vm.Set("loadlines", func(call goja.FunctionCall) goja.Value {
		name := valueString(call.Argument(0))
		if !vmFileHasRight(rights, name, 'r') {
			return vm.ToValue([]string{})
		}
		lines, err := loadVMLines(root, name)
		if err != nil {
			return vm.ToValue([]string{})
		}
		return vm.ToValue(lines)
	})
	vm.Set("savestring", func(call goja.FunctionCall) goja.Value {
		name := valueString(call.Argument(0))
		if !vmFileHasRight(rights, name, 'w') {
			return vm.ToValue(false)
		}
		err := saveVMString(root, name, valueString(call.Argument(1)), saveMode(call.Argument(2)))
		return vm.ToValue(err == nil)
	})
	vm.Set("savelines", func(call goja.FunctionCall) goja.Value {
		name := valueString(call.Argument(0))
		if !vmFileHasRight(rights, name, 'w') {
			return vm.ToValue(false)
		}
		err := saveVMLines(root, name, valueLines(call.Argument(1)), saveMode(call.Argument(2)))
		return vm.ToValue(err == nil)
	})
	vm.Set("deletefile", func(call goja.FunctionCall) goja.Value {
		name := valueString(call.Argument(0))
		if !vmFileHasRight(rights, name, 'w') {
			return vm.ToValue(false)
		}
		path, err := resolveVMFile(root, name)
		ok := err == nil && os.Remove(path) == nil
		return vm.ToValue(ok)
	})
	vm.Set("savelog2", func(call goja.FunctionCall) goja.Value {
		name := filepath.ToSlash(filepath.Join("logs", valueString(call.Argument(0))))
		if !vmFileHasRight(rights, name, 'w') {
			return vm.ToValue(false)
		}
		err := saveVMString(root, name, valueString(call.Argument(1))+"\n", true)
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
		proto.Set("addarray", func(call goja.FunctionCall) goja.Value {
			values := arrayValues(call.This)
			values = append(values, arrayValues(call.Argument(0))...)
			replaceArrayValues(vm, call.This.ToObject(vm), values)
			return call.This
		})
		proto.Set("insert", func(call goja.FunctionCall) goja.Value {
			values := arrayValues(call.This)
			index := int(valueInt(call.Argument(0)))
			if index < 0 {
				index = 0
			}
			if index > len(values) {
				index = len(values)
			}
			values = append(values[:index], append([]any{call.Argument(1).Export()}, values[index:]...)...)
			replaceArrayValues(vm, call.This.ToObject(vm), values)
			return call.This
		})
		proto.Set("replace", func(call goja.FunctionCall) goja.Value {
			values := arrayValues(call.This)
			index := int(valueInt(call.Argument(0)))
			if index >= 0 && index < len(values) {
				values[index] = call.Argument(1).Export()
				replaceArrayValues(vm, call.This.ToObject(vm), values)
			}
			return call.This
		})
		proto.Set("index", func(call goja.FunctionCall) goja.Value {
			values := arrayValues(call.This)
			needle := fmt.Sprint(call.Argument(0).Export())
			for i, value := range values {
				if fmt.Sprint(value) == needle {
					return vm.ToValue(i)
				}
			}
			return vm.ToValue(-1)
		})
		proto.Set("indices", func(call goja.FunctionCall) goja.Value {
			values := arrayValues(call.This)
			needle := fmt.Sprint(call.Argument(0).Export())
			var indices []int
			for i, value := range values {
				if fmt.Sprint(value) == needle {
					indices = append(indices, i)
				}
			}
			return vm.ToValue(indices)
		})
		proto.Set("delete", func(call goja.FunctionCall) goja.Value {
			values := arrayValues(call.This)
			index := int(valueInt(call.Argument(0)))
			if index >= 0 && index < len(values) {
				values = append(values[:index], values[index+1:]...)
				replaceArrayValues(vm, call.This.ToObject(vm), values)
			}
			return call.This
		})
		proto.Set("remove", func(call goja.FunctionCall) goja.Value {
			values := arrayValues(call.This)
			needle := call.Argument(0).Export()
			for i, value := range values {
				if fmt.Sprint(value) == fmt.Sprint(needle) {
					values = append(values[:i], values[i+1:]...)
					replaceArrayValues(vm, call.This.ToObject(vm), values)
					break
				}
			}
			return call.This
		})
		proto.Set("clear", func(call goja.FunctionCall) goja.Value {
			replaceArrayValues(vm, call.This.ToObject(vm), nil)
			return call.This
		})
		proto.Set("sortascending", func(call goja.FunctionCall) goja.Value {
			values := arrayValues(call.This)
			sort.SliceStable(values, func(i, j int) bool { return fmt.Sprint(values[i]) < fmt.Sprint(values[j]) })
			replaceArrayValues(vm, call.This.ToObject(vm), values)
			return call.This
		})
		proto.Set("sortdescending", func(call goja.FunctionCall) goja.Value {
			values := arrayValues(call.This)
			sort.SliceStable(values, func(i, j int) bool { return fmt.Sprint(values[i]) > fmt.Sprint(values[j]) })
			replaceArrayValues(vm, call.This.ToObject(vm), values)
			return call.This
		})
		proto.Set("sortbyvalue", func(call goja.FunctionCall) goja.Value {
			values := arrayValues(call.This)
			name := valueString(call.Argument(0))
			sortType := strings.ToLower(valueString(call.Argument(1)))
			ascending := true
			if len(call.Arguments) > 2 && !goja.IsUndefined(call.Argument(2)) {
				ascending = call.Argument(2).ToBoolean()
			}
			sort.SliceStable(values, func(i, j int) bool {
				if sortType == "float" || sortType == "double" || sortType == "int" {
					left := valueFloat(vm.ToValue(arrayMemberValue(values[i], name)))
					right := valueFloat(vm.ToValue(arrayMemberValue(values[j], name)))
					if ascending {
						return left < right
					}
					return left > right
				}
				left := fmt.Sprint(arrayMemberValue(values[i], name))
				right := fmt.Sprint(arrayMemberValue(values[j], name))
				if ascending {
					return left < right
				}
				return left > right
			})
			replaceArrayValues(vm, call.This.ToObject(vm), values)
			return call.This
		})
		proto.Set("insertarray", func(call goja.FunctionCall) goja.Value {
			values := arrayValues(call.This)
			insert := arrayValues(call.Argument(1))
			index := int(valueInt(call.Argument(0)))
			if index < 0 {
				index = 0
			}
			if index > len(values) {
				index = len(values)
			}
			values = append(values[:index], append(insert, values[index:]...)...)
			replaceArrayValues(vm, call.This.ToObject(vm), values)
			return call.This
		})
		proto.Set("subarray", func(call goja.FunctionCall) goja.Value {
			values := arrayValues(call.This)
			start := int(valueInt(call.Argument(0)))
			if start < 0 {
				start = 0
			}
			if start > len(values) {
				start = len(values)
			}
			end := len(values)
			if len(call.Arguments) > 1 && !goja.IsUndefined(call.Argument(1)) {
				length := int(valueInt(call.Argument(1)))
				if length < 0 {
					length = 0
				}
				end = start + length
				if end > len(values) {
					end = len(values)
				}
			}
			return vm.ToValue(values[start:end])
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
		proto.Set("startswith", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(strings.HasPrefix(valueString(call.This), valueString(call.Argument(0))))
		})
		proto.Set("ends", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(strings.HasSuffix(valueString(call.This), valueString(call.Argument(0))))
		})
		proto.Set("endswith", func(call goja.FunctionCall) goja.Value {
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
		filtered := files[:0]
		for _, file := range files {
			if vmFileHasRight(rights, file, 'r') {
				filtered = append(filtered, file)
			}
		}
		return vm.ToValue(filtered)
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

func vmFileHasRight(entries []string, name string, right rune) bool {
	if len(entries) == 0 {
		return true
	}
	name = filepath.ToSlash(strings.TrimLeft(strings.TrimSpace(valueStringLiteral(name)), "/"))
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, ":") {
		return false
	}
	for _, entry := range entries {
		rights := "r"
		pattern := strings.TrimSpace(entry)
		if parts := strings.SplitN(pattern, " ", 2); len(parts) == 2 {
			rights = strings.ToLower(strings.TrimSpace(parts[0]))
			pattern = strings.TrimSpace(parts[1])
		}
		pattern = filepath.ToSlash(strings.TrimLeft(pattern, "/"))
		if vmFilePatternMatch(pattern, name) && strings.ContainsRune(rights, right) {
			return true
		}
	}
	return false
}

func vmFilePatternMatch(pattern, name string) bool {
	pattern = filepath.ToSlash(strings.TrimLeft(strings.TrimSpace(pattern), "/"))
	name = filepath.ToSlash(strings.TrimLeft(strings.TrimSpace(valueStringLiteral(name)), "/"))
	matched, err := path.Match(pattern, name)
	return err == nil && matched
}

func desEncrypt(key, text string) ([]byte, error) {
	block, err := des.NewCipher(desKey(key))
	if err != nil {
		return nil, err
	}
	data := pkcs5Pad([]byte(text), block.BlockSize())
	out := make([]byte, len(data))
	for i := 0; i < len(data); i += block.BlockSize() {
		block.Encrypt(out[i:i+block.BlockSize()], data[i:i+block.BlockSize()])
	}
	return out, nil
}

func desDecrypt(key string, data []byte) ([]byte, error) {
	block, err := des.NewCipher(desKey(key))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("invalid DES data")
	}
	out := make([]byte, len(data))
	for i := 0; i < len(data); i += block.BlockSize() {
		block.Decrypt(out[i:i+block.BlockSize()], data[i:i+block.BlockSize()])
	}
	return pkcs5Unpad(out, block.BlockSize())
}

func desKey(key string) []byte {
	out := make([]byte, 8)
	copy(out, []byte(key))
	return out
}

func pkcs5Pad(data []byte, size int) []byte {
	pad := size - len(data)%size
	out := append([]byte(nil), data...)
	for i := 0; i < pad; i++ {
		out = append(out, byte(pad))
	}
	return out
}

func pkcs5Unpad(data []byte, size int) ([]byte, error) {
	if len(data) == 0 || len(data)%size != 0 {
		return nil, fmt.Errorf("invalid padding")
	}
	pad := int(data[len(data)-1])
	if pad <= 0 || pad > size || pad > len(data) {
		return nil, fmt.Errorf("invalid padding")
	}
	for _, value := range data[len(data)-pad:] {
		if int(value) != pad {
			return nil, fmt.Errorf("invalid padding")
		}
	}
	return data[:len(data)-pad], nil
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

func arrayValues(value goja.Value) []any {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}
	exported := value.Export()
	switch typed := exported.(type) {
	case []any:
		return append([]any(nil), typed...)
	case []string:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	default:
		return []any{typed}
	}
}

func arrayMemberValue(value any, name string) any {
	switch typed := value.(type) {
	case map[string]any:
		return typed[name]
	default:
		return ""
	}
}

func newGS2Array(dims []int) []any {
	if len(dims) == 0 {
		return []any{}
	}
	out := make([]any, dims[0])
	if len(dims) == 1 {
		return out
	}
	for i := range out {
		out[i] = newGS2Array(dims[1:])
	}
	return out
}

func replaceArrayValues(vm *goja.Runtime, obj *goja.Object, values []any) {
	oldLen := int(valueInt(obj.Get("length")))
	for i := 0; i < oldLen; i++ {
		_ = obj.Delete(strconv.Itoa(i))
	}
	for i, value := range values {
		obj.Set(strconv.Itoa(i), value)
	}
	obj.Set("length", len(values))
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
	id, _ := strconv.ParseUint(values["id"], 10, 16)
	return PlayerContext{ID: uint16(id), Account: values["account"], Nick: firstNonEmpty(values["nick"], values["nickname"]), Nickname: firstNonEmpty(values["nickname"], values["nick"]), Level: values["level"], Flags: flags, Rights: splitCSV(values["rights"]), Folders: splitLines(values["folders"])}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func splitLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	parts := strings.Split(value, "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func installNPCObjects(vm *goja.Runtime, result *Result, contexts []NPCContext, tracked *[]scriptNPCObject) {
	for _, context := range contexts {
		obj := npcObject(vm, result, context)
		*tracked = append(*tracked, scriptNPCObject{id: context.ID, name: context.Name, obj: obj, initial: cloneAnyMap(context.This)})
		if context.Name != "" && regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(context.Name) {
			vm.Set(context.Name, obj)
		}
	}
}

func npcObject(vm *goja.Runtime, result *Result, context NPCContext) *goja.Object {
	obj := objectFromAnyMap(vm, context.This)
	obj.Set("id", context.ID)
	obj.Set("name", context.Name)
	obj.Set("toString", func(call goja.FunctionCall) goja.Value { return vm.ToValue(context.Name) })
	for _, name := range scriptFunctionNames(context.Script) {
		functionName := name
		obj.Set(functionName, func(call goja.FunctionCall) goja.Value {
			npcCall := NPCFunctionCall{ID: context.ID, Name: context.Name, Function: functionName}
			for _, arg := range call.Arguments {
				npcCall.Args = append(npcCall.Args, valueString(arg))
			}
			result.NPCFunctionCalls = append(result.NPCFunctionCalls, npcCall)
			return goja.Undefined()
		})
	}
	return obj
}

func playerObject(vm *goja.Runtime, result *Result, player PlayerContext, players *[]scriptPlayerObject) *goja.Object {
	obj := vm.NewObject()
	obj.Set("id", player.ID)
	obj.Set("account", player.Account)
	obj.Set("nick", firstNonEmpty(player.Nick, player.Nickname))
	obj.Set("nickname", firstNonEmpty(player.Nickname, player.Nick))
	obj.Set("level", player.Level)
	obj.Set("toString", func(call goja.FunctionCall) goja.Value { return vm.ToValue(player.Account) })
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
	obj.Set("sendtorc", func(call goja.FunctionCall) goja.Value {
		if message := valueString(call.Argument(0)); message != "" {
			result.PlayerRCMessages = append(result.PlayerRCMessages, PlayerMessage{Account: player.Account, Message: message})
		}
		return goja.Undefined()
	})
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
	obj.Set("addWeapon", func(call goja.FunctionCall) goja.Value {
		addPlayerWeapon(result, player.Account, valueString(call.Argument(0)), true)
		return goja.Undefined()
	})
	obj.Set("removeweapon", func(call goja.FunctionCall) goja.Value {
		addPlayerWeapon(result, player.Account, valueString(call.Argument(0)), false)
		return goja.Undefined()
	})
	obj.Set("removeWeapon", func(call goja.FunctionCall) goja.Value {
		addPlayerWeapon(result, player.Account, valueString(call.Argument(0)), false)
		return goja.Undefined()
	})
	obj.Set("join", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	obj.Set("hasrightflag", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(playerHasRightFlag(player, valueString(call.Argument(0))))
	})
	obj.Set("hasright", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(playerHasFolderRight(player, valueString(call.Argument(0)), valueString(call.Argument(1))))
	})
	*players = append(*players, scriptPlayerObject{account: player.Account, client: client, clientr: clientr, initialClient: clientFlags, initialClientr: clientrFlags})
	return obj
}

func playerHasRightFlag(player PlayerContext, name string) bool {
	name = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(name, "-", "")))
	for _, right := range player.Rights {
		if strings.EqualFold(strings.ReplaceAll(right, "-", ""), name) {
			return true
		}
	}
	return false
}

func playerHasFolderRight(player PlayerContext, rights, name string) bool {
	for _, entry := range player.Folders {
		fields := strings.Fields(entry)
		if len(fields) < 2 || !strings.Contains(fields[0], rights) {
			continue
		}
		if vmFilePatternMatch(fields[1], name) {
			return true
		}
	}
	return false
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

func collectNPCFlags(result *Result, npcs []scriptNPCObject) {
	for _, npc := range npcs {
		if npc.obj == nil || npc.id == 0 {
			continue
		}
		for _, key := range npc.obj.Keys() {
			if key == "id" || key == "name" {
				continue
			}
			value := npc.obj.Get(key)
			if _, ok := goja.AssertFunction(value); ok {
				continue
			}
			if fmt.Sprint(npc.initial[key]) != valueString(value) {
				result.NPCFlags = append(result.NPCFlags, NPCFlag{ID: npc.id, Name: key, Value: valueString(value)})
			}
		}
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

func scriptFunctionNames(script string) []string {
	matches := regexp.MustCompile(`(?i)\bfunction\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`).FindAllStringSubmatch(script, -1)
	names := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := match[1]
		key := strings.ToLower(name)
		if !seen[key] {
			seen[key] = true
			names = append(names, name)
		}
	}
	return names
}

func cloneAnyMap(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
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
		out := make([]any, 0, len(parts))
		for i := range parts {
			out = append(out, typedGS2Value(parts[i]))
		}
		return out
	}
	return typedGS2Value(value)
}

func typedGS2Value(value string) any {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "true") {
		return true
	}
	if strings.EqualFold(value, "false") {
		return false
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
