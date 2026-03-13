package engine

import (
	"fmt"
	"math/rand"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// RollFunc is a function that evaluates a dice expression (e.g., "1d20") and returns the total.
type RollFunc func(dice string) int

// LuaEvaluator wraps a GopherLua environment configured for manifest formula evaluation.
type LuaEvaluator struct {
	L        *lua.LState
	rollFunc RollFunc
}

// NewLuaEvaluator creates a sandboxed Lua environment.
func NewLuaEvaluator(rollFunc RollFunc) (*LuaEvaluator, error) {
	if rollFunc == nil {
		rollFunc = defaultRoll
	}

	L := lua.NewState(lua.Options{
		SkipOpenLibs: true, // We manually open only safe libs
	})

	// Open safe standard libraries
	for _, pair := range []struct {
		n string
		f lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		if err := L.CallByParam(lua.P{
			Fn:      L.NewFunction(pair.f),
			NRet:    0,
			Protect: true,
		}, lua.LString(pair.n)); err != nil {
			L.Close()
			return nil, fmt.Errorf("failed to load lua lib %s: %w", pair.n, err)
		}
	}

	ev := &LuaEvaluator{L: L, rollFunc: rollFunc}

	// Register Go functions
	L.SetGlobal("roll", L.NewFunction(ev.luaRoll))

	// Register event helper functions — each returns a tagged table { _event = "...", ... }
	registerEventHelpers(L)

	// Add dynamic fallback for "is_*_active" functions to return false.
	mt := L.NewTable()
	L.SetField(mt, "__index", L.NewFunction(func(L2 *lua.LState) int {
		key := L2.CheckString(2)
		fmt.Printf("[DEBUG] __index called for key: %s\n", key)
		if strings.HasPrefix(key, "is_") && strings.HasSuffix(key, "_active") {
			L2.Push(L2.NewFunction(func(L3 *lua.LState) int {
				L3.Push(lua.LBool(false))
				return 1
			}))
			return 1
		}
		L2.Push(lua.LNil)
		return 1
	}))
	L.SetMetatable(L.GetGlobal("_G"), mt)

	return ev, nil
}

// Close releases the Lua state resources.
func (ev *LuaEvaluator) Close() {
	ev.L.Close()
}

// luaRoll exposes the rollFunc to Lua scripts: roll("1d20") -> number
func (ev *LuaEvaluator) luaRoll(L *lua.LState) int {
	dice := L.CheckString(1)
	result := ev.rollFunc(dice)
	L.Push(lua.LNumber(result))
	return 1
}

// registerEventHelpers registers typed Lua functions that return tagged tables.
func registerEventHelpers(L *lua.LState) {
	// loop(name, active) -> { _event = "loop", name = name, active = active }
	L.SetGlobal("loop", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("loop"))
		t.RawSetString("name", L.Get(1))
		t.RawSetString("active", L.Get(2))
		L.Push(t)
		return 1
	}))

	// loop_order(name, ascending) -> { _event = "loop_order", name = name, ascending = ascending }
	L.SetGlobal("loop_order", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("loop_order"))
		t.RawSetString("name", L.Get(1))
		t.RawSetString("ascending", L.Get(2))
		L.Push(t)
		return 1
	}))

	// loop_value(name, value) -> { _event = "loop_value", name = name, value = value }
	L.SetGlobal("loop_value", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("loop_value"))
		t.RawSetString("name", L.Get(1))
		t.RawSetString("value", L.Get(2))
		L.Push(t)
		return 1
	}))

	// add_actor(id_or_list) -> { _event = "add_actor", actors = id_or_list }
	L.SetGlobal("add_actor", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("add_actor"))
		t.RawSetString("actors", L.Get(1))
		L.Push(t)
		return 1
	}))

	// ask(target, ...options) -> { _event = "ask", target = target, options = {...} }
	L.SetGlobal("ask", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("ask"))
		t.RawSetString("target", L.Get(1))
		opts := L.NewTable()
		for i := 2; i <= L.GetTop(); i++ {
			opts.RawSetInt(i-1, L.Get(i))
		}
		t.RawSetString("options", opts)
		L.Push(t)
		return 1
	}))

	// condition(cond) -> { _event = "condition", condition = cond, add = true }
	L.SetGlobal("condition", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("condition"))
		t.RawSetString("condition", L.Get(1))
		t.RawSetString("add", lua.LTrue)
		L.Push(t)
		return 1
	}))

	// remove_condition(cond) -> { _event = "condition", condition = cond, add = false }
	L.SetGlobal("remove_condition", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("condition"))
		t.RawSetString("condition", L.Get(1))
		t.RawSetString("add", lua.LFalse)
		L.Push(t)
		return 1
	}))

	// spend(key, [amount]) -> { _event = "spend", key = key, amount = amount }
	L.SetGlobal("spend", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("spend"))
		t.RawSetString("key", L.Get(1))
		amt := L.Get(2)
		if amt == lua.LNil {
			amt = lua.LNumber(1)
		}
		t.RawSetString("amount", amt)
		L.Push(t)
		return 1
	}))

	// set_attr(section, key, val) -> { _event = "set_attr", section = section, key = key, value = val }
	L.SetGlobal("set_attr", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("set_attr"))
		t.RawSetString("section", L.Get(1))
		t.RawSetString("key", L.Get(2))
		t.RawSetString("value", L.Get(3))
		L.Push(t)
		return 1
	}))

	// contest(roll_value) -> { _event = "contest", value = roll_value }
	L.SetGlobal("contest", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("contest"))
		t.RawSetString("value", L.Get(1))
		L.Push(t)
		return 1
	}))

	// check(passed) -> { _event = "check", passed = passed }
	L.SetGlobal("check_result", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("check"))
		t.RawSetString("passed", L.Get(1))
		L.Push(t)
		return 1
	}))

	// hint(msg) -> { _event = "hint", message = msg }
	L.SetGlobal("hint", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("hint"))
		t.RawSetString("message", L.Get(1))
		L.Push(t)
		return 1
	}))

	// metadata(key, val) -> { _event = "metadata", key = key, value = val }
	L.SetGlobal("metadata", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("metadata"))
		t.RawSetString("key", L.Get(1))
		t.RawSetString("value", L.Get(2))
		L.Push(t)
		return 1
	}))

	// emit(type, payload) -> { _event = type, payload = payload }
	L.SetGlobal("emit", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", L.Get(1))
		t.RawSetString("payload", L.Get(2))
		L.Push(t)
		return 1
	}))

	// next_turn(loop_name) -> { _event = "next_turn", name = loop_name }
	L.SetGlobal("next_turn", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		t.RawSetString("_event", lua.LString("next_turn"))
		t.RawSetString("name", L.Get(1))
		L.Push(t)
		return 1
	}))
}

// Eval evaluates a Lua expression (string) or closure (*lua.LFunction) against the given context.
func (ev *LuaEvaluator) Eval(formula any, ctx map[string]any) (any, error) {
	// Inject the context into the Lua globals
	for k, v := range ctx {
		ev.L.SetGlobal(k, goValueToLua(ev.L, v))
	}

	// Always clear the globals after execution to prevent leaked state between calls
	defer func() {
		for k := range ctx {
			ev.L.SetGlobal(k, lua.LNil)
		}
	}()

	switch f := formula.(type) {
	case string:
		// Option A: Evaluate string formula
		script := "return type(is_encounter_start_active) .. ' : ' .. tostring(is_encounter_start_active)\n"
		if err := ev.L.DoString(script); err == nil {
			lv := ev.L.Get(-1)
			ev.L.Pop(1)
			fmt.Printf("[DEBUG] Evaluating script: %q, type of is_encounter_start_active: %s\n", f, lv.String())
		}

		script = "return " + f
		if err := ev.L.DoString(script); err != nil {
			return nil, fmt.Errorf("Lua eval error: %w", err)
		}
		// Extract result (the return value is on top of the stack)
		lv := ev.L.Get(-1)
		ev.L.Pop(1)
		return luaValueToGo(lv), nil

	case *lua.LFunction:
		// Option B: Call the closure
		if err := ev.L.CallByParam(lua.P{
			Fn:      f,
			NRet:    1,
			Protect: true,
		}); err != nil {
			return nil, fmt.Errorf("Lua call error: %w", err)
		}
		lv := ev.L.Get(-1)
		ev.L.Pop(1)
		return luaValueToGo(lv), nil

	case bool, int, float64:
		// Literal values
		return f, nil

	default:
		return nil, fmt.Errorf("unsupported formula type: %T", formula)
	}
}

// LoadManifestLua reads and executes a manifest.lua file, extracting the commands and restrictions.
// NOTE: This must be called at engine startup to populate the globals.
func (ev *LuaEvaluator) LoadManifestLua(path string) (*Manifest, error) {
	m := &Manifest{
		Commands: make(map[string]CommandDef),
	}

	if err := ev.L.DoFile(path); err != nil {
		return nil, fmt.Errorf("failed to load manifest.lua: %w", err)
	}

	// Read commands table
	cmdsVal := ev.L.GetGlobal("commands")
	if cmdsTbl, ok := cmdsVal.(*lua.LTable); ok {
		cmdsTbl.ForEach(func(k, v lua.LValue) {
			cmdName := k.String()
			// v is a table representing CommandDef
			if t, ok := v.(*lua.LTable); ok {
				cmdDef := parseCommandDefFromLua(t)
				m.Commands[cmdName] = cmdDef
			}
		})
	} else {
		return nil, fmt.Errorf("manifest.lua must define a 'commands' table")
	}

	// Read restrictions table
	resVal := ev.L.GetGlobal("restrictions")
	if resTbl, ok := resVal.(*lua.LTable); ok {
		m.Restrictions = parseRestrictionsFromLua(resTbl)
	}

	return m, nil
}

func parseCommandDefFromLua(t *lua.LTable) CommandDef {
	def := CommandDef{}
	if name := t.RawGetString("name"); name != lua.LNil {
		def.Name = name.String()
	}
	if hint := t.RawGetString("hint"); hint != lua.LNil {
		def.Hint = hint.String()
	}
	if help := t.RawGetString("help"); help != lua.LNil {
		def.Help = help.String()
	}
	if errStr := t.RawGetString("error"); errStr != lua.LNil {
		def.Error = errStr.String()
	}

	if params := t.RawGetString("params"); params != lua.LNil {
		if pt, ok := params.(*lua.LTable); ok {
			for i := 1; i <= pt.Len(); i++ {
				p := pt.RawGetInt(i)
				if paramTbl, ok := p.(*lua.LTable); ok {
					pd := ParamDef{
						Name:     paramTbl.RawGetString("name").String(),
						Type:     paramTbl.RawGetString("type").String(),
						Required: paramTbl.RawGetString("required") == lua.LTrue,
					}
					def.Params = append(def.Params, pd)
				}
			}
		}
	}

	def.Prereq = parsePrereqStepsFromLua(t.RawGetString("prereq"))
	def.Game = parseCommandPhaseFromLua(t.RawGetString("game"))
	def.Targets = parseCommandPhaseFromLua(t.RawGetString("targets"))
	def.Actor = parseCommandPhaseFromLua(t.RawGetString("actor"))

	return def
}

func parsePrereqStepsFromLua(val lua.LValue) []PrereqStep {
	if t, ok := val.(*lua.LTable); ok {
		var steps []PrereqStep
		for i := 1; i <= t.Len(); i++ {
			p := t.RawGetInt(i)
			if stepTbl, ok := p.(*lua.LTable); ok {
				ps := PrereqStep{
					Name:  stepTbl.RawGetString("name").String(),
					Error: stepTbl.RawGetString("error").String(),
				}
				f := stepTbl.RawGetString("value")
				if str, ok := f.(lua.LString); ok {
					ps.Value = string(str)
				} else if fn, ok := f.(*lua.LFunction); ok {
					ps.Value = fn
				} else if bl, ok := f.(lua.LBool); ok {
					ps.Value = bool(bl)
				}

				steps = append(steps, ps)
			}
		}
		return steps
	}
	return nil
}

func parseCommandPhaseFromLua(val lua.LValue) CommandPhase {
	var phase CommandPhase

	if t, ok := val.(*lua.LTable); ok {
		// 1. Iterate array portion for Steps
		stepsTbl := t
		if sVal := t.RawGetString("steps"); sVal != lua.LNil {
			if sTbl, ok := sVal.(*lua.LTable); ok {
				stepsTbl = sTbl
			}
		}

		for i := 1; i <= stepsTbl.Len(); i++ {
			p := stepsTbl.RawGetInt(i)
			if stepTbl, ok := p.(*lua.LTable); ok {
				gs := GameStep{
					Name: stepTbl.RawGetString("name").String(),
				}

				f := stepTbl.RawGetString("value")
				if str, ok := f.(lua.LString); ok {
					gs.Value = string(str)
				} else if fn, ok := f.(*lua.LFunction); ok {
					gs.Value = fn
				} else if bl, ok := f.(lua.LBool); ok {
					gs.Value = bool(bl)
				}

				phase.Steps = append(phase.Steps, gs)
			}
		}

		// 2. Look for "hooks" key for Hooks
		if hooksVal := t.RawGetString("hooks"); hooksVal != lua.LNil {
			if hooksTbl, ok := hooksVal.(*lua.LTable); ok {
				for i := 1; i <= hooksTbl.Len(); i++ {
					p := hooksTbl.RawGetInt(i)
					if hookTbl, ok := p.(*lua.LTable); ok {
						hd := HookDef{
							Name: hookTbl.RawGetString("name").String(),
							Type: hookTbl.RawGetString("type").String(),
						}

						f := hookTbl.RawGetString("value")
						if str, ok := f.(lua.LString); ok {
							hd.Value = string(str)
						} else if fn, ok := f.(*lua.LFunction); ok {
							hd.Value = fn
						} else if bl, ok := f.(lua.LBool); ok {
							hd.Value = bool(bl)
						}

						phase.Hooks = append(phase.Hooks, hd)
					}
				}
			}
		}
	}
	return phase
}

func parseRestrictionsFromLua(t *lua.LTable) Restrictions {
	var r Restrictions

	if adj := t.RawGetString("adjudication"); adj != lua.LNil {
		if adjTbl, ok := adj.(*lua.LTable); ok {
			if cmds := adjTbl.RawGetString("commands"); cmds != lua.LNil {
				if cmdTbl, ok := cmds.(*lua.LTable); ok {
					for i := 1; i <= cmdTbl.Len(); i++ {
						r.Adjudication.Commands = append(r.Adjudication.Commands, cmdTbl.RawGetInt(i).String())
					}
				}
			}
		}
	}

	if gm := t.RawGetString("gm_commands"); gm != lua.LNil {
		if gmTbl, ok := gm.(*lua.LTable); ok {
			for i := 1; i <= gmTbl.Len(); i++ {
				r.GMCommands = append(r.GMCommands, gmTbl.RawGetInt(i).String())
			}
		}
	}

	return r
}

// goValueToLua converts native Go values to Lua values.
func goValueToLua(L *lua.LState, val any) lua.LValue {
	if val == nil {
		return lua.LNil
	}
	switch v := val.(type) {
	case string:
		return lua.LString(v)
	case int:
		return lua.LNumber(v)
	case int64:
		return lua.LNumber(v)
	case float64:
		return lua.LNumber(v)
	case bool:
		return lua.LBool(v)
	case []any:
		tbl := L.NewTable()
		for i, item := range v {
			tbl.RawSetInt(i+1, goValueToLua(L, item))
		}
		return tbl
	case []string:
		tbl := L.NewTable()
		for i, item := range v {
			tbl.RawSetInt(i+1, lua.LString(item))
		}
		return tbl
	case map[string]any:
		tbl := L.NewTable()
		for mk, mv := range v {
			tbl.RawSetString(mk, goValueToLua(L, mv))
		}
		return tbl
	case map[string]int:
		tbl := L.NewTable()
		for mk, mv := range v {
			tbl.RawSetString(mk, lua.LNumber(mv))
		}
		return tbl
	case map[string]string:
		tbl := L.NewTable()
		for mk, mv := range v {
			tbl.RawSetString(mk, lua.LString(mv))
		}
		return tbl
	case func() any:
		return L.NewFunction(func(L2 *lua.LState) int {
			res := v()
			if res == nil {
				L2.Push(lua.LNil)
			} else {
				L2.Push(goValueToLua(L2, res))
			}
			return 1
		})
	// Handling *lua.LFunction for nested function support if needed
	case lua.LValue:
		return v
	default:
		// Attempt fallback via fmt
		return lua.LString(fmt.Sprintf("%v", v))
	}
}

// luaValueToGo converting Lua values to Go interface{} equivalents.
func luaValueToGo(lv lua.LValue) any {
	switch lv.Type() {
	case lua.LTString:
		return string(lv.(lua.LString))
	case lua.LTNumber:
		// Convert to int if it's an integer value, otherwise float64
		num := float64(lv.(lua.LNumber))
		if num == float64(int(num)) {
			return int(num)
		}
		return num
	case lua.LTBool:
		return bool(lv.(lua.LBool))
	case lua.LTTable:
		tbl := lv.(*lua.LTable)
		// Check if it's an array-like table
		if tbl.MaxN() > 0 {
			arr := make([]any, tbl.MaxN())
			for i := 1; i <= tbl.MaxN(); i++ {
				arr[i-1] = luaValueToGo(tbl.RawGetInt(i))
			}
			return arr
		}
		// Otherwise map-like table
		m := make(map[string]any)
		tbl.ForEach(func(key, val lua.LValue) {
			m[key.String()] = luaValueToGo(val)
		})
		return m
	case lua.LTNil:
		return nil
	default:
		return lv.String()
	}
}

// BuildContext constructs the Lua evaluation context from the current game state.
// We keep this function structure similar to the CEL one, but now it generates a Go map
// that is injected via goValueToLua in Eval().
func BuildContext(state *GameState, actor *Entity, target *Entity, params map[string]any, gameRes, targetRes, actorRes map[string]any) map[string]any {
	ctx := map[string]any{
		"command":       params,
		"game":          gameRes,
		"targets":       targetRes,
		"actor_results": actorRes,
		"metadata":      state.Metadata,
	}

	if actor != nil {
		ctx["actor"] = entityToMap(actor)
	} else {
		ctx["actor"] = map[string]any{}
	}

	if target != nil {
		ctx["target"] = entityToMap(target)
	} else {
		ctx["target"] = map[string]any{}
	}

	// Inject loop state and current actor functions
	for name, loop := range state.Loops {
		active := loop.Active
		ctx["is_"+name+"_active"] = func() any { return active }
	}

	ctx["current_actor"] = func() any {
		for _, loop := range state.Loops {
			if loop.Active && len(loop.Actors) > 0 {
				sorted := sortedActors(loop)
				if loop.Current >= 0 && loop.Current < len(sorted) {
					return sorted[loop.Current]
				}
			}
		}
		return nil
	}

	return ctx
}

// entityToMap converts an Entity to a map[string]any suitable for Lua evaluation.
func entityToMap(e *Entity) map[string]any {
	if e == nil {
		return nil
	}
	return map[string]any{
		"id":            e.ID,
		"name":          e.Name,
		"types":         e.Types,
		"classes":       e.Classes,
		"stats":         e.Stats,
		"resources":     e.Resources,
		"spent":         e.Spent,
		"conditions":    e.Conditions,
		"proficiencies": e.Proficiencies,
		"statuses":      e.Statuses,
		"inventory":     e.Inventory,
	}
}

func defaultRoll(dice string) int {
	var count, sides int
	if _, err := fmt.Sscanf(dice, "%dd%d", &count, &sides); err != nil || sides <= 0 {
		return 0
	}
	total := 0
	for i := 0; i < count; i++ {
		total += rand.Intn(sides) + 1
	}
	return total
}
