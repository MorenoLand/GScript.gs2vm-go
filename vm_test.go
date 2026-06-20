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
