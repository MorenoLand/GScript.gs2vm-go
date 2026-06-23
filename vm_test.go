package gs2vm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunEchoesParamsAndPlayerAccount(t *testing.T) {
	result := Run(Config{
		ScriptName: "-gr_movement",
		EventName:  "onActionServerside",
		Script: `function onActionServerside() {
			echo("test" SPC params[0] SPC player.account);
		}`,
		Params: []string{"from clientside"},
		Player: map[string]string{"account": "moondeath"},
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "test from clientside moondeath" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunSupportsOneLineFunctionBodies(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script:    `function onCreated() foo(), echo("kek"), clientr.foo = "bar"; function foo() echo("tits");`,
		Player:    map[string]string{"account": "moondeath"},
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 2 || result.Output[0] != "tits" || result.Output[1] != "kek" {
		t.Fatalf("Run output = %#v", result.Output)
	}
	if !hasPlayerFlag(result.PlayerFlags, "moondeath", "clientr.foo", "bar") {
		t.Fatalf("missing clientr flag: %#v", result.PlayerFlags)
	}
}

func TestRunCollectsTopLevelNPCPropsWithoutEventFunction(t *testing.T) {
	result := Run(Config{
		ScriptName: "Control-NPC",
		EventName:  "onCreated",
		NPCID:      7,
		Script:     `chat = 1;`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.NPCActions) != 1 || result.NPCActions[0].Props["chat"] != "1" {
		t.Fatalf("NPC actions = %#v", result.NPCActions)
	}
}

func TestRunShowCharacterAppliesDefaultCharacterProps(t *testing.T) {
	result := Run(Config{
		ScriptName: "Control-NPC",
		EventName:  "onCreated",
		NPCID:      7,
		Script: `function onCreated() {
			showcharacter();
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.NPCActions) == 0 || result.NPCActions[0].Props["image"] != "#c#" || result.NPCActions[0].Props["ani"] != "idle" {
		t.Fatalf("NPC actions = %#v", result.NPCActions)
	}
}

func TestRunShowCharacterAllowsIndexedColorWrites(t *testing.T) {
	result := Run(Config{
		ScriptName: "Control-NPC",
		EventName:  "onCreated",
		NPCID:      7,
		Script: `function onCreated() {
			showcharacter();
			this.colors[0] = "orange";
			this.colors[1] = "white";
			this.colors[2] = "blue";
			this.colors[3] = "red";
			this.colors[4] = "black";
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.NPCActions) == 0 || result.NPCActions[0].Props["colors"] != "orange,white,blue,red,black" {
		t.Fatalf("NPC actions = %#v", result.NPCActions)
	}
}

func TestRunPlayerLifecycleEventPassesPlayerObjectArgument(t *testing.T) {
	result := Run(Config{
		EventName: "onPlayerLogin",
		Player:    map[string]string{"account": "bob", "nickname": "Bob"},
		Script: `function onPlayerLogin(pl) {
			echo("+" SPC pl.account SPC params[0].account);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "+ bob bob" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunSupportsControlNPCLoginScript(t *testing.T) {
	script := `function onPlayerLogin(temp.pl) {
  sendtonc(temp.pl.nick SPC "("@temp.pl.account@") "@(temp.pl.hasrightflag("warptoxy")? "(staff)":"")@": has logged online!");
  if (temp.pl.hasrightflag("warptoxy")) {
    temp.sw = {
                "-Staff/Toolbar",
                "-Staff/Level_Editor",
                "-Staff/NPC_Editor",
                "-Staff/Gani_Editor",
                "-Staff/Link_Editor",
                "-Staff/Console",
                "-Staff/Gui_Explorer",
                "-den/etiles"
              };
    for (temp.n: temp.sw) temp.pl.addWeapon(n);
  }
  temp.pl.addWeapon("-Core");
  if (temp.pl.account.starts("pc:")) {
    temp.pl.addWeapon("-ActivateClient");
  }
  temp.pl.join("player");
  temp.pl.join("func_core");
  this.fart = true;
}

function onPlayerLogout(temp.pl) {
  sendtonc(temp.pl.nick SPC "("@temp.pl.account@"): has gone offline!");
}

function onCreated() {
  for (temp.p: allplayers) {
    if (temp.p.account == "Graal6973523") p.guild = "Manager";
    if (temp.p.account == "Graal6973527") p.guild = "Waste of Space";
  }
}
function getFile(_file) return base64encode(temp.mp.loadstring(findfiles(_file, true)[0]) ? mp : mp);

function onRCChat(cmd, data) {
echo(params);
  temp._rt = cmd.pos("newrc") > -1? "RC3" : cmd.pos("sublimerc") > -1? (cmd.pos("grc") > -1? "SublimeRC (grc)" : "SublimeRC (py)") : nil;
  if (_rt) printf("RC Detection: %s (%s)", _rt, data);
  if(cmd == "file") player.sendtorc(format("file:%s:%s",getextension(data),getFile(data)));
}`
	result := Run(Config{
		EventName: "onPlayerLogin",
		Script:    script,
		Player:    map[string]string{"account": "moondeath", "nick": "Moon", "rights": "warptoxy,NPC-Control"},
	})
	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.NCMessages) != 1 || result.NCMessages[0] != "Moon (moondeath) (staff): has logged online!" {
		t.Fatalf("NCMessages = %#v", result.NCMessages)
	}
	if !hasPlayerWeapon(result.PlayerWeapons, "moondeath", "-Core", true) || !hasPlayerWeapon(result.PlayerWeapons, "moondeath", "-Staff/Toolbar", true) {
		t.Fatalf("PlayerWeapons = %#v", result.PlayerWeapons)
	}
}

func TestRunExposesServerFlagsAndOptions(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			echo(serverr.poopybutthole SPC serveroptions.staff[1]);
		}`,
		ServerFlags:   map[string]string{"serverr.poopybutthole": "true"},
		ServerOptions: map[string]string{"staff": "(Manager),moondeath"},
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "true moondeath" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunSupportsIndexedTypedServerFlagValues(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			if (serverr.poopybutthole[0] == true) {
				echo(serverr.poopybutthole[1]);
			}
		}`,
		ServerFlags: map[string]string{"serverr.poopybutthole": "true,2"},
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "2" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunStopsAtMaxLoopLimit(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			maxlooplimit = 3;
			for (temp.i = 0; temp.i < 10; temp.i++) {
				echo(temp.i);
			}
		}`,
	})

	if result.Err == "" || !strings.Contains(result.Err, "maxlooplimit") {
		t.Fatalf("Run err = %q output=%#v", result.Err, result.Output)
	}
}

func TestRunCollectsMutableServerFlags(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			server.foo = "bar";
			serverr.secret = "yes";
			delete server.old;
			serveroptions.staff = "changed";
		}`,
		ServerFlags:   map[string]string{"server.old": "1"},
		ServerOptions: map[string]string{"staff": "original"},
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.ServerFlags) != 3 {
		t.Fatalf("Run ServerFlags = %#v", result.ServerFlags)
	}
	flags := map[string]string{}
	deleted := map[string]bool{}
	for _, flag := range result.ServerFlags {
		flags[flag.Name] = flag.Value
		deleted[flag.Name] = flag.Deleted
	}
	if flags["server.foo"] != "bar" || flags["serverr.secret"] != "yes" {
		t.Fatalf("Run ServerFlags = %#v", result.ServerFlags)
	}
	if !deleted["server.old"] {
		t.Fatalf("Run ServerFlags missing deleted server.old: %#v", result.ServerFlags)
	}
	if _, ok := flags["serveroptions.staff"]; ok {
		t.Fatalf("serveroptions write was returned as mutable flag: %#v", result.ServerFlags)
	}
}

func TestRunCollectsTriggerClient(t *testing.T) {
	result := Run(Config{
		ScriptName: "-gr_movement",
		EventName:  "onActionServerside",
		Script: `function onActionServerside() {
			triggerclient("gui", name, "kek");
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.ClientTriggers) != 1 {
		t.Fatalf("Run ClientTriggers = %#v", result.ClientTriggers)
	}
	trigger := result.ClientTriggers[0]
	if trigger.Kind != "gui" || trigger.Name != "-gr_movement" || len(trigger.Args) != 1 || trigger.Args[0] != "kek" {
		t.Fatalf("trigger = %#v", trigger)
	}
}

func TestRunCollectsTSocketActions(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			this.socket = new TSocket("HTTPSocket");
			this.socket.packagedelimiter = "\n" @ char(13) @ "\n";
			this.socket.bind(1234, false);
			this.socket.send("hello");
			this.socket.close();
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.SocketActions) != 3 {
		t.Fatalf("Run SocketActions = %#v", result.SocketActions)
	}
	if result.SocketActions[0].Action != "bind" || result.SocketActions[0].Name != "HTTPSocket" || result.SocketActions[0].Port != 1234 || result.SocketActions[0].PackageDelimiter != "\n\r\n" {
		t.Fatalf("bind action = %#v", result.SocketActions[0])
	}
	if result.SocketActions[1].Action != "send" || result.SocketActions[1].Data != "hello" {
		t.Fatalf("send action = %#v", result.SocketActions[1])
	}
	if result.SocketActions[2].Action != "close" {
		t.Fatalf("close action = %#v", result.SocketActions[2])
	}
}

func TestRunSupportsDottedSocketEvents(t *testing.T) {
	result := Run(Config{
		EventName: "HTTPSocket.onBind",
		Script: `function HTTPSocket.onBind() {
			echo(this.name @ " bound");
		}`,
		This: map[string]any{"name": "HTTPSocket"},
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "HTTPSocket bound" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunSupportsTempParameterFunctions(t *testing.T) {
	result := Run(Config{
		EventName: "onReceiveDataPackage",
		Params:    []string{"GET / HTTP/1.1"},
		Script: `function onReceiveDataPackage(temp.str) {
			echo(temp.str);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "GET / HTTP/1.1" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunCollectsScheduledEvents(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			this.scheduleevent(1, "onBindSockets");
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.ScheduledEvents) != 1 || result.ScheduledEvents[0].Event != "onBindSockets" || result.ScheduledEvents[0].Delay != 1 {
		t.Fatalf("Run ScheduledEvents = %#v", result.ScheduledEvents)
	}
}

func TestRunSupportsCamelCaseScheduleEventAndBareEventNames(t *testing.T) {
	script := `function onCreated() {
			scheduleEvent(1, "Kek");
			this.scheduleEvent(2, "onOther");
		}
		function onKek() { echo("foo"); }
		function onOther() { echo("bar"); }`
	result := Run(Config{
		EventName: "onCreated",
		Script:    script,
	})
	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.ScheduledEvents) != 2 || result.ScheduledEvents[0].Event != "Kek" || result.ScheduledEvents[1].Event != "onOther" {
		t.Fatalf("Run ScheduledEvents = %#v", result.ScheduledEvents)
	}
	next := Run(Config{EventName: result.ScheduledEvents[0].Event, Script: script})
	if next.Err != "" {
		t.Fatalf("next Run err = %q", next.Err)
	}
	if len(next.Output) != 1 || next.Output[0] != "foo" {
		t.Fatalf("next output = %#v", next.Output)
	}
}

func TestRunCollectsNPCActions(t *testing.T) {
	result := Run(Config{
		NPCID:     25,
		EventName: "onActionBob",
		Params:    []string{"kek"},
		Script: `function onActionBob(prm1) {
			setshape(1, 32, 48);
			move(2.5, -1.5, 0.5, 24);
			chat = "Bob param" SPC prm1 SPC params[0];
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.NPCActions) != 3 {
		t.Fatalf("Run NPCActions = %#v", result.NPCActions)
	}
	if result.NPCActions[0].ID != 25 || result.NPCActions[0].ShapeType != 1 || result.NPCActions[0].Width != 32 || result.NPCActions[0].Height != 48 {
		t.Fatalf("shape action = %#v", result.NPCActions[0])
	}
	if result.NPCActions[1].MoveDX != 2.5 || result.NPCActions[1].MoveDY != -1.5 || result.NPCActions[1].MoveTime != 0.5 || result.NPCActions[1].MoveOptions != 24 {
		t.Fatalf("move action = %#v", result.NPCActions[1])
	}
	if result.NPCActions[2].Chat != "Bob param kek kek" {
		t.Fatalf("chat action = %#v", result.NPCActions[2])
	}
	if len(result.ScheduledEvents) != 1 || result.ScheduledEvents[0].Event != "onMovementFinished" || result.ScheduledEvents[0].Delay != 0.5 {
		t.Fatalf("move scheduled events = %#v", result.ScheduledEvents)
	}
}

func TestRunCollectsCurrentNPCPropertiesAndStateActions(t *testing.T) {
	result := Run(Config{
		NPCID:     25,
		EventName: "onCreated",
		Script: `function onCreated() {
			this.image = "block.png";
			this.chat = "hi";
			this.dir = 1;
			this.head = "head1.png";
			this.body = "body.png";
			this.bombs = 4;
			hide();
			dontblock();
			drawoverplayer();
			canbecarried();
			cannotbepushed();
			destroy();
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.NPCActions) != 1 {
		t.Fatalf("Run NPCActions = %#v", result.NPCActions)
	}
	action := result.NPCActions[0]
	if action.Props["image"] != "block.png" || action.Props["chat"] != "hi" || action.Props["dir"] != "1" || action.Props["head"] != "head1.png" || action.Props["body"] != "body.png" || action.Props["bombs"] != "4" {
		t.Fatalf("action props = %#v", action.Props)
	}
	if !action.HasVisFlags || action.VisFlags != 3 || !action.HasBlockFlags || action.BlockFlags != 1 || !action.Destroy {
		t.Fatalf("action state = %#v", action)
	}
	if action.Flags["canbecarried"] != "true" || action.Flags["canbepushed"] != "false" {
		t.Fatalf("action flags = %#v", action.Flags)
	}
}

func TestRunFindNPCObjectsAndCalls(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		NPCs: []NPCContext{
			{ID: 10000, Name: "Control-NPC", This: map[string]any{"old": "1"}, Script: `function callMe(value, flag) { echo(value SPC flag); }`},
			{ID: 10001, Name: "DenDB", Script: `function callMe(value, flag) { echo(value SPC flag); }`},
		},
		Script: `function onCreated() {
			temp.foo = findnpc("Control-NPC");
			temp.foo.kek = true;
			findnpcbyid(10000).old = "2";
			("Control-NPC").other = "yes";
			DenDB.callMe("Crazy", true);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.NPCFlags) != 3 {
		t.Fatalf("NPCFlags = %#v", result.NPCFlags)
	}
	wantFlags := map[string]string{"kek": "true", "old": "2", "other": "yes"}
	for _, flag := range result.NPCFlags {
		if flag.ID != 10000 || flag.Name == "" || wantFlags[flag.Name] != flag.Value {
			t.Fatalf("unexpected NPC flag %#v all=%#v", flag, result.NPCFlags)
		}
		delete(wantFlags, flag.Name)
	}
	if len(wantFlags) != 0 {
		t.Fatalf("missing NPC flags %#v all=%#v", wantFlags, result.NPCFlags)
	}
	if len(result.NPCFunctionCalls) != 1 || result.NPCFunctionCalls[0].Name != "DenDB" || result.NPCFunctionCalls[0].Function != "callMe" || result.NPCFunctionCalls[0].Args[0] != "Crazy" || result.NPCFunctionCalls[0].Args[1] != "true" {
		t.Fatalf("NPCFunctionCalls = %#v", result.NPCFunctionCalls)
	}
}

func TestRunUsesScriptThisAsObjectAndString(t *testing.T) {
	result := Run(Config{
		ScriptName: "-gr_movement",
		EventName:  "onCreated",
		Player:     map[string]string{"account": "Denveous"},
		Players:    []PlayerContext{{Account: "Denveous"}},
		Script: `function onCreated() {
			this.kek = true;
			findplayer("Denveous").addweapon(this);
			echo(this.name SPC this);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if result.This["kek"] != true {
		t.Fatalf("Run this = %#v", result.This)
	}
	if len(result.PlayerWeapons) != 1 || result.PlayerWeapons[0].Account != "Denveous" || result.PlayerWeapons[0].Name != "-gr_movement" || !result.PlayerWeapons[0].Add {
		t.Fatalf("PlayerWeapons = %#v", result.PlayerWeapons)
	}
	if len(result.Output) != 1 || result.Output[0] != "-gr_movement -gr_movement" {
		t.Fatalf("Output = %#v", result.Output)
	}
}

func TestRunSupportsMoreParityHelpers(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			temp.word = "abcdef";
			echo(temp.word.substring(2, 3) SPC ("b" in {"a", "b", "c"}) SPC "  Hi ".trim().lower() SPC strlen("abcd"));
			temp.foo = "bar";
			echo(foo.pos("r") SPC "abcdef".startswith("abc") SPC "abcdef".endswith("def") SPC foo.starts("ba") SPC foo.ends("ar"));
			echo(temp.word.substring(3));
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 3 || result.Output[0] != "cde true hi 4" || result.Output[1] != "2 true true true true" || result.Output[2] != "def" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunSupportsCommonGlobalParityHelpers(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			echo(abs(-2) SPC ceil(1.2) SPC floor(1.8) SPC int(strtofloat("4.9")));
			echo(strequals("A", "A") SPC strcontains("abcdef", "cd") SPC contains("abcdef", "ef"));
			temp.foo = "Hello WORLD";
			echo(startswith("abcdef", "ab") SPC endswith("abcdef", "ef") SPC uppercase("hi") SPC lowercase("HI") SPC foo.upper() SPC foo.lower());
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 3 {
		t.Fatalf("Run output = %#v", result.Output)
	}
	if result.Output[0] != "2 2 1 4" || result.Output[1] != "true true true" || result.Output[2] != "true true HI hi HELLO WORLD hello world" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunSupportsStringMethodReceiversOnLiteralsAndExpressions(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Player:    map[string]string{"account": "moondeath"},
		Script: `function onCreated() {
			echo("HEY THERE".lower());
			echo(("HEY THERE " @ player.account).lower());
			echo(("hey there " @ player.account).upper());
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 3 || result.Output[0] != "hey there" || result.Output[1] != "hey there moondeath" || result.Output[2] != "HEY THERE MOONDEATH" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunSupportsDynamicPropertyNames(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Player:    map[string]string{"account": "moondeath"},
		Script: `function onCreated() {
			this.("kek_" @ player.account) = true;
			echo(this.("kek_" @ player.account));
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "true" {
		t.Fatalf("Run output = %#v", result.Output)
	}
	if result.This["kek_moondeath"] != true {
		t.Fatalf("Run this = %#v", result.This)
	}
}

func TestRunSupportsTabAndNLConcatenators(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			echo("a" TAB "b");
			echo("c" NL "d");
			echo(TAB @ NL);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 3 || result.Output[0] != "a\tb" || result.Output[1] != "c\nd" || result.Output[2] != "\t\n" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunExposesPlayersWeaponsServersAndNPCWarp(t *testing.T) {
	result := Run(Config{
		NPCID:     25,
		EventName: "onCreated",
		Players: []PlayerContext{
			{Account: "moondeath", Nick: "*moondeath"},
			{Account: "guest", Nick: "guest"},
		},
		Weapons: []WeaponContext{{Name: "-gr_movement", Image: "wbomb1.png"}},
		Servers: []ServerContext{{Name: "Orion-Go", Type: "Gold", PlayerCount: 3, Language: "English", Description: "Go Code GServer", URL: "https://example.test", Version: "Custom version", GameVersions: "2.220,6.037", Latency: 42}},
		Script: `function onCreated() {
			echo(allplayers.length SPC allplayers[0].account SPC weapons[0].name SPC servers[0].name SPC servers[0].players SPC servers[0].latency);
			warpto("test.nw", 30, 31);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "2 moondeath -gr_movement Orion-Go 3 42" {
		t.Fatalf("Run output = %#v", result.Output)
	}
	if len(result.NPCActions) != 1 || result.NPCActions[0].WarpLevel != "test.nw" || result.NPCActions[0].WarpX != 30 || result.NPCActions[0].WarpY != 31 {
		t.Fatalf("Run NPCActions = %#v", result.NPCActions)
	}
}

func TestRunBase64AndScreenGlobals(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			echo(base64decode(base64encode("kek")) SPC screenwidth SPC screenheight);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "kek 1024 1024" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunImageHelpers(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			showimg(200, "block.png", 4, 5);
			findimg(200).rotation = 3;
			echo(findimg(200).image SPC findimg(200).rotation SPC getimgwidth("block.png") SPC getimgheight("block.png"));
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "block.png 3 1 1" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunIgnoresClientsideBlock(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() { echo("server"); }
//#CLIENTSIDE
function onCreated() { echo("client"); }`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "server" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunCapturesPlayerClientFlags(t *testing.T) {
	result := Run(Config{
		EventName:   "onCreated",
		Player:      map[string]string{"account": "moondeath"},
		PlayerFlags: map[string]string{"client.old": "1"},
		Script: `function onCreated() {
			client.foo = "bar";
			clientr.secret = "ok";
			player.client.extra = "yes";
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if !hasPlayerFlag(result.PlayerFlags, "moondeath", "client.foo", "bar") {
		t.Fatalf("missing client flag: %#v", result.PlayerFlags)
	}
	if !hasPlayerFlag(result.PlayerFlags, "moondeath", "clientr.secret", "ok") {
		t.Fatalf("missing clientr flag: %#v", result.PlayerFlags)
	}
	if !hasPlayerFlag(result.PlayerFlags, "moondeath", "client.extra", "yes") {
		t.Fatalf("missing player.client alias flag: %#v", result.PlayerFlags)
	}
}

func TestRunFindPlayerSendPMAndFlags(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Players: []PlayerContext{
			{Account: "moondeath", Nickname: "*moondeath", Flags: map[string]string{"clientr.hp": "3"}},
		},
		Script: `function onCreated() {
			temp.pl = findplayer("moondeath");
			if (temp.pl != null) {
				temp.pl.clientr.hp = "2";
				temp.pl.sendpm("hey there");
				temp.pl.sendplayer("second");
				foo = temp.pl;
				foo.sendpm("third");
				findplayer("moondeath").sendpm("fourth");
			}
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if !hasPlayerFlag(result.PlayerFlags, "moondeath", "clientr.hp", "2") {
		t.Fatalf("missing findplayer flag update: %#v", result.PlayerFlags)
	}
	if len(result.PlayerMessages) != 4 || result.PlayerMessages[0].Account != "moondeath" || result.PlayerMessages[0].Message != "hey there" || result.PlayerMessages[1].Message != "second" || result.PlayerMessages[2].Message != "third" || result.PlayerMessages[3].Message != "fourth" {
		t.Fatalf("PlayerMessages = %#v", result.PlayerMessages)
	}
}

func TestRunSupportsBareSendPMCompatibility(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Player:    map[string]string{"account": "moondeath"},
		Script: `function onCreated() {
			sendpm(player.account, "aids");
			sendplayer(player.account, "second");
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.PlayerMessages) != 2 || result.PlayerMessages[0].Account != "moondeath" || result.PlayerMessages[0].Message != "aids" || result.PlayerMessages[1].Message != "second" {
		t.Fatalf("PlayerMessages = %#v", result.PlayerMessages)
	}
}

func TestRunTempAssignmentCreatesBareAliasForCurrentEvent(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Players:   []PlayerContext{{Account: "moondeath"}},
		Script: `function onCreated() {
			temp.foo = findplayer("moondeath");
			foo.sendpm("kek");
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.PlayerMessages) != 1 || result.PlayerMessages[0].Message != "kek" {
		t.Fatalf("PlayerMessages = %#v", result.PlayerMessages)
	}
}

func TestRunCollectsPlayerSetLevelActions(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Player:    map[string]string{"account": "moondeath"},
		Script: `function onCreated() {
			setlevel2("self.nw", 12, 13);
			player.setlevel("inside.nw");
			player.setlevel2("outside.nw", 30, 13.5);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.PlayerWarps) != 3 {
		t.Fatalf("PlayerWarps = %#v", result.PlayerWarps)
	}
	if result.PlayerWarps[0].Account != "moondeath" || result.PlayerWarps[0].Level != "self.nw" || result.PlayerWarps[0].X != 12 || result.PlayerWarps[0].Y != 13 {
		t.Fatalf("first warp = %#v", result.PlayerWarps[0])
	}
	if result.PlayerWarps[1].Account != "moondeath" || result.PlayerWarps[1].Level != "inside.nw" {
		t.Fatalf("second warp = %#v", result.PlayerWarps[1])
	}
	if result.PlayerWarps[2].Account != "moondeath" || result.PlayerWarps[2].Level != "outside.nw" || result.PlayerWarps[2].X != 30 || result.PlayerWarps[2].Y != 13.5 {
		t.Fatalf("third warp = %#v", result.PlayerWarps[2])
	}
}

func TestRunCollectsPlayerWeaponActions(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Player:    map[string]string{"account": "moondeath"},
		Players:   []PlayerContext{{Account: "bob"}},
		Script: `function onCreated() {
			addweapon("-self");
			removeweapon("-old");
			findplayer("bob").addweapon("-bob");
			findplayer("bob").removeweapon("-gone");
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.PlayerWeapons) != 4 {
		t.Fatalf("PlayerWeapons = %#v", result.PlayerWeapons)
	}
	want := []PlayerWeapon{{Account: "moondeath", Name: "-self", Add: true}, {Account: "moondeath", Name: "-old"}, {Account: "bob", Name: "-bob", Add: true}, {Account: "bob", Name: "-gone"}}
	for i := range want {
		if result.PlayerWeapons[i] != want[i] {
			t.Fatalf("PlayerWeapons[%d] = %#v want %#v", i, result.PlayerWeapons[i], want[i])
		}
	}
}

func TestRunExposesPlayerCollectionsAndHelpers(t *testing.T) {
	root := filepath.Join(".", ".tmp", "gs2vm-file-helpers")
	_ = os.RemoveAll(root)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	result := Run(Config{
		EventName: "onCreated",
		FileRoot:  root,
		Player:    map[string]string{"account": "self", "dir": "3"},
		Players:   []PlayerContext{{ID: 7, Account: "Denveous", Dir: 1}, {ID: 8, Account: "bob", Dir: 2}},
		Script: `function onCreated() {
			for (temp.p: allplayers) if (p.account == "Denveous") echo(p.id);
			echo(players[1].account);
			echo(players[1].id);
			echo(player.dir SPC players[0].dir SPC players[1].dir);
			temp.encrypted = base64encode(des_encrypt("12345678", "Hello World"));
			echo(des_decrypt("12345678", base64decode(temp.encrypted)));
			savelog2("kek.txt", "hi");
			savestring("delete-me.txt", "x", 0);
			echo(deletefile("delete-me.txt"));
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	want := []string{"7", "bob", "8", "3 1 2", "Hello World", "true"}
	if len(result.Output) != len(want) {
		t.Fatalf("Output = %#v", result.Output)
	}
	for i := range want {
		if result.Output[i] != want[i] {
			t.Fatalf("Output[%d] = %q want %q all=%#v", i, result.Output[i], want[i], result.Output)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "logs", "kek.txt"))
	if err != nil || string(data) != "hi\n" {
		t.Fatalf("log file data=%q err=%v", string(data), err)
	}
	if _, err := os.Stat(filepath.Join(root, "delete-me.txt")); !os.IsNotExist(err) {
		t.Fatalf("delete-me stat err=%v", err)
	}
}

func TestRunFileHelpersRespectRights(t *testing.T) {
	root := filepath.Join(".", ".tmp", "gs2vm-file-rights")
	_ = os.RemoveAll(root)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	if err := os.MkdirAll(filepath.Join(root, "data"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "data", "read.txt"), []byte("ok"), 0644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	result := Run(Config{
		EventName:  "onCreated",
		FileRoot:   root,
		FileRights: []string{"r data/read.txt", "w data/write.txt", "w logs/*"},
		Script: `function onCreated() {
			echo(loadstring("data/read.txt"));
			echo(loadstring("data/write.txt"));
			echo(savestring("data/read.txt", "no", 0));
			echo(savestring("data/write.txt", "yes", 0));
			echo(deletefile("data/read.txt"));
			savelog2("kek.txt", "hi");
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	want := []string{"ok", "", "false", "true", "false"}
	if len(result.Output) != len(want) {
		t.Fatalf("Output = %#v", result.Output)
	}
	for i := range want {
		if result.Output[i] != want[i] {
			t.Fatalf("Output[%d] = %q want %q all=%#v", i, result.Output[i], want[i], result.Output)
		}
	}
	if data, err := os.ReadFile(filepath.Join(root, "data", "write.txt")); err != nil || string(data) != "yes" {
		t.Fatalf("write data=%q err=%v", string(data), err)
	}
	if data, err := os.ReadFile(filepath.Join(root, "logs", "kek.txt")); err != nil || string(data) != "hi\n" {
		t.Fatalf("log data=%q err=%v", string(data), err)
	}
}

func TestRunTranslatesGS2ConcatenatorsEnumsAndArrays(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `enum {
			WALK,
			ATTACK,
			DEAD
		}
		function onCreated() {
			temp.s = "a";
			s @= "b";
			this.health = {5, 7};
			if (ATTACK == 1 && DEAD == 2) echo(s @ TAB @ this.health[1] @ NL @ "ok");
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "ab\t7\nok" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunSupportsArrayParityHelpers(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			temp.items = {"b", "a", "c"};
			temp.items.sortascending();
			echo(temp.items[0] SPC temp.items[1] SPC temp.items[2]);
			temp.items.sortdescending();
			echo(temp.items[0] SPC temp.items[1] SPC temp.items[2]);
			temp.items.insertarray(1, {"x", "y"});
			temp.part = temp.items.subarray(1, 3);
			echo(temp.items[0] SPC temp.items[1] SPC temp.items[2] SPC temp.items[3] SPC temp.items[4]);
			echo(temp.part[0] SPC temp.part[1] SPC temp.part[2]);
			trace("traced" SPC temp.part.size());
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	want := []string{"a b c", "c b a", "c x y b a", "x y b", "traced 3"}
	if len(result.Output) != len(want) {
		t.Fatalf("Run output = %#v", result.Output)
	}
	for i := range want {
		if result.Output[i] != want[i] {
			t.Fatalf("Run output[%d] = %q want %q all=%#v", i, result.Output[i], want[i], result.Output)
		}
	}
}

func TestRunSupportsArrayMutationParityHelpers(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			temp.items = {"a", "b", "c", "d"};
			temp.items.remove("c");
			echo(temp.items.size() SPC temp.items[0] SPC temp.items[1] SPC temp.items[2]);
			temp.items.delete(1);
			echo(temp.items.size() SPC temp.items[0] SPC temp.items[1]);
			temp.items.clear();
			echo(temp.items.size());
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	want := []string{"3 a b d", "2 a d", "0"}
	if len(result.Output) != len(want) {
		t.Fatalf("Run output = %#v", result.Output)
	}
	for i := range want {
		if result.Output[i] != want[i] {
			t.Fatalf("Run output[%d] = %q want %q all=%#v", i, result.Output[i], want[i], result.Output)
		}
	}
}

func TestRunSupportsArraySortByValue(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			temp.items = new[3];
			temp.items[0] = new Object();
			temp.items[0].name = "b";
			temp.items[0].score = 2;
			temp.items[1] = new Object();
			temp.items[1].name = "a";
			temp.items[1].score = 10;
			temp.items[2] = new Object();
			temp.items[2].name = "c";
			temp.items[2].score = 1;
			temp.items.sortbyvalue("name", "string", true);
			echo(temp.items[0].name SPC temp.items[1].name SPC temp.items[2].name);
			temp.items.sortbyvalue("score", "float", false);
			echo(temp.items[0].score SPC temp.items[1].score SPC temp.items[2].score);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	want := []string{"a b c", "10 2 1"}
	if len(result.Output) != len(want) {
		t.Fatalf("Run output = %#v", result.Output)
	}
	for i := range want {
		if result.Output[i] != want[i] {
			t.Fatalf("Run output[%d] = %q want %q all=%#v", i, result.Output[i], want[i], result.Output)
		}
	}
}

func TestRunSupportsMoreArrayObjectMethods(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			temp.items = {"a", "b", "c", "b"};
			temp.items.addarray({"d", "e"});
			echo(temp.items[4] SPC temp.items[5] SPC temp.items.size());
			temp.items.insert(1, "x");
			echo(temp.items[0] SPC temp.items[1] SPC temp.items[2] SPC temp.items.size());
			temp.items.replace(2, "y");
			echo(temp.items[0] SPC temp.items[1] SPC temp.items[2]);
			echo(temp.items.index("b"));
			temp.idxs = temp.items.indices("b");
			echo(temp.idxs.size() SPC temp.idxs[0] SPC temp.idxs[1]);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	want := []string{"d e 6", "a x b 7", "a x y", "4", "1 4 undefined"}
	if len(result.Output) != len(want) {
		t.Fatalf("Run output = %#v", result.Output)
	}
	for i := range want {
		if result.Output[i] != want[i] {
			t.Fatalf("Run output[%d] = %q want %q all=%#v", i, result.Output[i], want[i], result.Output)
		}
	}
}

func TestRunTranslatesConstsAndNewArrays(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			const kek = "true";
			temp.bar = {"foo", "bar"};
			temp.foo = new[temp.bar.size()];
			temp.foo[1] = "kek";
			echo(kek SPC temp.foo.size() SPC temp.foo[1]);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "true 2 kek" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunSupportsScreenshotIssueParity(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			temp.grid = new[2][3];
			temp.grid[1][2] = "ok";
			echo(temp.grid.size() SPC temp.grid[1].size() SPC temp.grid[1][2]);
			temp.i = 0;
			do {
				temp.i++;
			} while (temp.i < 3);
			echo(temp.i);
			echo(hideimgs(1, 2) SPC keycode(65));
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	want := []string{"2 3 ok", "3", "0 65"}
	if len(result.Output) != len(want) {
		t.Fatalf("Run output = %#v", result.Output)
	}
	for i := range want {
		if result.Output[i] != want[i] {
			t.Fatalf("Run output[%d] = %q want %q all=%#v", i, result.Output[i], want[i], result.Output)
		}
	}
}

func TestRunTranslatesForLoops(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			For (temp.i = 0; temp.i < 3; temp.i++) {
				echo(temp.i);
			}
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 3 || result.Output[0] != "0" || result.Output[1] != "1" || result.Output[2] != "2" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunTranslatesForEachLoops(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			temp.foo = {"bar", "echo"};
			for (temp.bar: temp.foo) {
				echo(bar);
			}
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 2 || result.Output[0] != "bar" || result.Output[1] != "echo" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunTranslatesDynamicFunctionCalls(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			temp.foo = "bar";
			(@foo)();
			(@foo)("again");
		}
		function bar(value) {
			if (value == null) value = "called";
			echo(value);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 2 || result.Output[0] != "called" || result.Output[1] != "again" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunSupportsStringAndLineFileGlobals(t *testing.T) {
	root := testFileRoot(t)
	result := Run(Config{
		EventName: "onCreated",
		FileRoot:  root,
		Script: `function onCreated() {
			savestring("data/text.txt", "hello", 0);
			savestring("data/text.txt", " world", 1);
			savelines("data/lines.txt", {"one", "two"}, 0);
			temp.loaded = loadlines("data/lines.txt");
			echo(loadstring("data/text.txt") SPC temp.loaded[0] SPC temp.loaded[1]);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "hello world one two" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunSupportsStringAndLineFileMethods(t *testing.T) {
	root := testFileRoot(t)
	result := Run(Config{
		EventName: "onCreated",
		FileRoot:  root,
		Script: `function onCreated() {
			"hello".savestring("method/text.txt", 0);
			temp.lines = {"alpha", "beta"};
			temp.lines.savelines("method/lines.txt", 0);
			temp.loaded = {};
			temp.loaded.loadlines("method/lines.txt");
			echo(loadstring("method/text.txt") SPC temp.loaded[0] SPC temp.loaded[1]);
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.Output) != 1 || result.Output[0] != "hello alpha beta" {
		t.Fatalf("Run output = %#v", result.Output)
	}
}

func TestRunPersistsThisButNotTempThroughHostState(t *testing.T) {
	first := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			temp.once = "gone";
			this.saved = "kept";
		}`,
	})
	if first.Err != "" {
		t.Fatalf("first Run err = %q", first.Err)
	}
	second := Run(Config{
		EventName: "onActionServerside",
		This:      first.This,
		Script: `function onActionServerside() {
			if (this.saved == "kept" && typeof temp.once == "undefined") echo("ok");
		}`,
	})
	if second.Err != "" {
		t.Fatalf("second Run err = %q", second.Err)
	}
	if len(second.Output) != 1 || second.Output[0] != "ok" {
		t.Fatalf("second output = %#v this=%#v", second.Output, second.This)
	}
}

func TestRunCollectsGlobalNPCChatFromEvent(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		NPCID:     7,
		Script:    `function onCreated() chat = 1;`,
	})
	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.NPCActions) != 1 || result.NPCActions[0].Chat != "1" {
		t.Fatalf("NPCActions = %#v", result.NPCActions)
	}
}

func TestRunSupportsCaseInsensitiveHostFunctionAliases(t *testing.T) {
	result := Run(Config{
		ScriptName: "-gr_movement",
		EventName:  "onCreated",
		Player:     map[string]string{"account": "moondeath"},
		Players:    []PlayerContext{{Account: "moondeath", Nick: "moondeath"}},
		Script:     `function onCreated() findPlayer("moondeath").addWeapon(this);`,
	})
	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.PlayerWeapons) != 1 || result.PlayerWeapons[0].Account != "moondeath" || result.PlayerWeapons[0].Name != "-gr_movement" {
		t.Fatalf("PlayerWeapons = %#v", result.PlayerWeapons)
	}
}

func TestRunSetShapeAppliesShapeSize(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		NPCID:     7,
		Script:    `function onCreated() setshape(1, 32, 32);`,
	})
	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.NPCActions) != 1 || result.NPCActions[0].ShapeType != 1 || result.NPCActions[0].Width != 32 || result.NPCActions[0].Height != 32 {
		t.Fatalf("NPCActions = %#v", result.NPCActions)
	}
}

func TestRunRCChatHelpersAndRights(t *testing.T) {
	root := testFileRoot(t)
	if err := os.WriteFile(filepath.Join(root, "script.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	result := Run(Config{
		EventName: "onRCChat",
		FileRoot:  root,
		FileRights: []string{
			"r script.txt",
		},
		Player: map[string]string{
			"account": "moondeath",
			"nick":    "Moon",
			"rights":  "warptoxy,NPC-Control",
			"folders": "r script.txt",
		},
		Script: `function getFile(file) return base64encode(temp.mp.loadstring(findfiles(file, true)[0]) ? mp : mp);
			function onRCChat(cmd, data) {
				if (cmd == "newrc" && player.hasrightflag("warptoxy")) printf("RC Detection: %s (%s)", "RC3", data);
				if (cmd == "file") player.sendtorc(format("file:%s:%s", getextension(data), getFile(data)));
			}`,
		Params: []string{"newrc", "2015.10.31"},
	})
	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.NCMessages) != 1 || result.NCMessages[0] != "RC Detection: RC3 (2015.10.31)" {
		t.Fatalf("NCMessages = %#v", result.NCMessages)
	}
	result = Run(Config{
		EventName: "onRCChat",
		FileRoot:  root,
		FileRights: []string{
			"r script.txt",
		},
		Player: map[string]string{
			"account": "moondeath",
			"folders": "r script.txt",
		},
		Script: `function getFile(file) return base64encode(temp.mp.loadstring(findfiles(file, true)[0]) ? mp : mp);
			function onRCChat(cmd, data) {
				if (cmd == "file") player.sendtorc(format("file:%s:%s", getextension(data), getFile(data)));
			}`,
		Params: []string{"file", "script.txt"},
	})
	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.PlayerRCMessages) != 1 || result.PlayerRCMessages[0].Account != "moondeath" || result.PlayerRCMessages[0].Message != "file:txt:aGVsbG8=" {
		t.Fatalf("PlayerRCMessages = %#v", result.PlayerRCMessages)
	}
}

func hasPlayerFlag(flags []PlayerFlag, account, name, value string) bool {
	for _, flag := range flags {
		if flag.Account == account && flag.Name == name && flag.Value == value {
			return true
		}
	}
	return false
}

func hasPlayerWeapon(weapons []PlayerWeapon, account, name string, add bool) bool {
	for _, weapon := range weapons {
		if weapon.Account == account && weapon.Name == name && weapon.Add == add {
			return true
		}
	}
	return false
}

func testFileRoot(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp(".", ".test-gs2-files-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Fatalf("cleanup %s: %v", root, err)
		}
	})
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
