package gs2vm

import (
	"os"
	"path/filepath"
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

func TestRunCollectsMutableServerFlags(t *testing.T) {
	result := Run(Config{
		EventName: "onCreated",
		Script: `function onCreated() {
			server.foo = "bar";
			serverr.secret = "yes";
			serveroptions.staff = "changed";
		}`,
		ServerFlags:   map[string]string{"server.old": "1"},
		ServerOptions: map[string]string{"staff": "original"},
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if len(result.ServerFlags) != 2 {
		t.Fatalf("Run ServerFlags = %#v", result.ServerFlags)
	}
	flags := map[string]string{}
	for _, flag := range result.ServerFlags {
		flags[flag.Name] = flag.Value
	}
	if flags["server.foo"] != "bar" || flags["serverr.secret"] != "yes" {
		t.Fatalf("Run ServerFlags = %#v", result.ServerFlags)
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

func hasPlayerFlag(flags []PlayerFlag, account, name, value string) bool {
	for _, flag := range flags {
		if flag.Account == account && flag.Name == name && flag.Value == value {
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
