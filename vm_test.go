package gs2vm

import "testing"

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
			}
		}`,
	})

	if result.Err != "" {
		t.Fatalf("Run err = %q", result.Err)
	}
	if !hasPlayerFlag(result.PlayerFlags, "moondeath", "clientr.hp", "2") {
		t.Fatalf("missing findplayer flag update: %#v", result.PlayerFlags)
	}
	if len(result.PlayerMessages) != 2 || result.PlayerMessages[0].Account != "moondeath" || result.PlayerMessages[0].Message != "hey there" || result.PlayerMessages[1].Message != "second" {
		t.Fatalf("PlayerMessages = %#v", result.PlayerMessages)
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
