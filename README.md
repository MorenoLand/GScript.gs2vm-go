# GScript.gs2vm-go

Native Go GS2 runtime VM backed by goja.

This module runs server-side GS2-style scripts for the Go GServer NPC-server runtime. It is not the clientside bytecode compiler.

## Script Model

- Code after `//#CLIENTSIDE` is stripped before server-side execution.
- `function onCreated()` runs when a script is updated or reloaded.
- `function onInitialized()` is for NPC-server startup initialization.
- Scheduled events call matching `on<EventName>()` handlers.
- Event parameters are available through `params[index]` and named function parameters.
- `temp.` variables are per-run temporary values.
- `this.` variables persist with the script object until the script is reloaded.

## Globals

- `player`
- `players`
- `allplayers`
- `weapons`
- `server`
- `serverr`
- `serveroptions`
- `params`
- `temp`
- `name`
- `maxlooplimit`
- `TAB`
- `NL`
- `NULL`
- `nil`
- `screenwidth`
- `screenheight`

## Functions

### Output

- `echo(value, ...)`
- `trace(value, ...)`
- `printf(format, ...)`
- `sendtorc(message)`
- `sendtonc(message)`

### Player Lookup And Messaging

- `findplayer(accountOrNick)`
- `findPlayer(accountOrNick)`
- `sendpm(account, message)`
- `sendPM(account, message)`
- `sendplayer(account, message)`

Player objects support:

- `sendpm(message)`
- `sendplayer(message)`
- `sendtorc(message)`
- `setlevel(level)`
- `setlevel2(level, x, y)`
- `addweapon(name)`
- `addWeapon(name)`
- `removeweapon(name)`
- `removeWeapon(name)`
- `join(className)`
- `hasrightflag(flagName)`
- `hasright(rights, fileName)`

### NPC Lookup And Events

- `findnpc(name)`
- `findNPC(name)`
- `findnpcbyid(id)`
- `findNPCByID(id)`
- `triggerclient(type, name, ...)`
- `triggerClient(type, name, ...)`

NPC objects can call public/custom functions by name. Level NPC triggeraction events currently include:

- `onAction<Name>`
- `onActionLeftMouse`
- `onActionRightMouse`
- `onActionMiddleMouse`
- `onActionDoubleMouse`
- `onPlayerTouchsMe`

### Current NPC

Inside an NPC script, these work as globals and as `this.` methods where applicable:

- `showcharacter()`
- `setshape(type, width, height)`
- `setshape2(width, height, tileTypes)`
- `warpto(level, x, y)`
- `move(dx, dy, time, options)`
- `hide()`
- `show()`
- `destroy()`
- `dontblock()`
- `blockagain()`
- `drawoverplayer()`
- `drawunderplayer()`
- `drawaslight()`
- `canbecarried()`
- `cannotbecarried()`
- `canbepulled()`
- `cannotbepulled()`
- `canbepushed()`
- `cannotbepushed()`
- `canwarp()`
- `canwarp2()`
- `cannotwarp()`

Current NPC properties collected after execution include:

- `image`
- `chat`
- `nick`
- `dir`
- `ani`
- `headimg`
- `bodyimg`
- `swordimg`
- `shieldimg`
- `horseimg`
- `colors`
- `hearts`
- `gralats`
- `arrows`
- `bombs`
- `glovepower`
- `swordpower`
- `shieldpower`
- `ap`
- `width`
- `height`

### Classes And Scheduling

- `loadclass(name)`
- `join(name)`
- `leave(name)`
- `scheduleevent(delay, event)`
- `scheduleEvent(delay, event)`

`this.scheduleevent(...)`, `this.scheduleEvent(...)`, `this.join(...)`, and `this.leave(...)` are also available.

### Strings And Arrays

Supported string methods include:

- `.substring(start[, length])`
- `.tokenize(delimiter)`
- `.pos(search)`
- `.starts(search)`
- `.startswith(search)`
- `.ends(search)`
- `.endswith(search)`
- `.trim()`
- `.lower()`
- `.upper()`
- `.savestring(file, mode)`
- `.loadstring(file)`

Supported array methods include:

- `.add(value)`
- `.addarray(values)`
- `.insert(index, value)`
- `.insertarray(index, values)`
- `.replace(index, value)`
- `.delete(index)`
- `.remove(value)`
- `.clear()`
- `.index(value)`
- `.indices(value)`
- `.subarray(start[, end])`
- `.sortascending()`
- `.sortdescending()`
- `.sortbyvalue(key[, ascending])`
- `.savelines(file, mode)`
- `.loadlines(file)`

### Files

- `loadstring(file)`
- `loadlines(file)`
- `savestring(file, value, mode)`
- `savelines(file, values, mode)`
- `deletefile(file)`
- `savelog2(file, text)`
- `findfiles(pattern, recursive)`
- `getextension(file)`

File access is controlled by the host-provided file rights.

### Encoding And Crypto

- `base64encode(value)`
- `base64decode(value)`
- `des_encrypt(key, value)`
- `des_decrypt(key, value)`

### Images And UI Helpers

- `showimg(index, image, x, y)`
- `findimg(index)`
- `hideimgs(start, count)`
- `getimgwidth(image)`
- `getimgheight(image)`
- `keycode(value)`
- `openurl(url)`

### Math And Conversion

- `int(value)`
- `float(value)`
- `double(value)`
- `strtofloat(value)`
- `abs(value)`
- `ceil(value)`
- `floor(value)`
- `sin(value)`
- `cos(value)`
- `tan(value)`
- `random(min, max)`
- `char(code)`
- `strlen(value)`
- `format(format, ...)`
- `strequals(a, b)`
- `strcontains(a, b)`
- `contains(a, b)`
- `startswith(a, b)`
- `endswith(a, b)`
- `uppercase(value)`
- `lowercase(value)`

### Sockets

- `new TSocket(name)`

Socket objects support:

- `bind(port, ssl)`
- `send(data)`
- `close()`
- `destroy()`
- `join(className)`

## Host Responsibilities

The embedding server provides players, NPCs, weapons, server flags, server options, file roots, file rights, sockets, and action dispatch. The VM returns actions and state changes; the host applies them to the actual game world.
