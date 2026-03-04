package engine

import (
	"fmt"
	"math/rand"

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
	// mod is no longer a Go function; the plan states it is part of manifest.lua

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
		script := "return " + f
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
	def.Game = parseGameStepsFromLua(t.RawGetString("game"))
	def.Targets = parseGameStepsFromLua(t.RawGetString("targets"))
	def.Actor = parseGameStepsFromLua(t.RawGetString("actor"))

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
				// Formula can be string or function
				f := stepTbl.RawGetString("formula")
				if str, ok := f.(lua.LString); ok {
					ps.Formula = string(str)
				} else if fn, ok := f.(*lua.LFunction); ok {
					ps.Formula = fn
				} else if bl, ok := f.(lua.LBool); ok {
					ps.Formula = bool(bl)
				}

				steps = append(steps, ps)
			}
		}
		return steps
	}
	return nil
}

func parseGameStepsFromLua(val lua.LValue) []GameStep {
	if t, ok := val.(*lua.LTable); ok {
		var steps []GameStep
		for i := 1; i <= t.Len(); i++ {
			p := t.RawGetInt(i)
			if stepTbl, ok := p.(*lua.LTable); ok {
				gs := GameStep{
					Name:  stepTbl.RawGetString("name").String(),
					Event: stepTbl.RawGetString("event").String(),
				}
				if loop := stepTbl.RawGetString("loop"); loop != lua.LNil {
					gs.Loop = loop.String()
				}

				f := stepTbl.RawGetString("formula")
				if str, ok := f.(lua.LString); ok {
					gs.Formula = string(str)
				} else if fn, ok := f.(*lua.LFunction); ok {
					gs.Formula = fn
				} else if bl, ok := f.(lua.LBool); ok {
					gs.Formula = bool(bl)
				}

				steps = append(steps, gs)
			}
		}
		return steps
	}
	return nil
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

	// Inject loop state (the old 'is_<name>_active')
	for name, loop := range state.Loops {
		ctx["is_"+name+"_active"] = loop.Active
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
