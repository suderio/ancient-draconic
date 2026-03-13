restrictions = {
    adjudication = {
        commands = { "grapple", "hide", "improvise" },
    },
    gm_commands = { "encounter_start", "encounter_end", "add_condition", "remove_condition" },
}

local _sizes_list = { "tiny", "small", "medium", "large", "huge", "gargantuan" }
local _skill_map = {
    athletics = "str",
    acrobatics = "dex",
    stealth = "dex",
    sleight_of_hand = "dex",
    perception = "wis",
    insight = "wis",
    survival = "wis",
    investigation = "int",
    arcana = "int",
    history = "int",
    religion = "int",
    nature = "int",
    medicine = "wis",
    persuasion = "cha",
    intimidation = "cha",
    performance = "cha",
    deception = "cha",
}

function mod(val)
    if not val then
        return -5
    end
    return math.floor(val / 2) - 5
end

function sizes(input)
    for i, name in ipairs(_sizes_list) do
        if name == input then
            return i
        end
    end
    return 0
end

function skill_to_ability(skill)
    return _skill_map[skill] or "str"
end

commands = {
    encounter_start = {
        name = "encounter start",
        params = {
            { name = "with", type = "list<target>", required = false },
        },
        prereq = {
            {
                name = "check_conflict",
                value = function()
                    return not is_encounter_start_active()
                end,
                error = "an encounter is already active. End it first",
            },
        },
        hint = "Encounter has started. Roll initiative for all actors.",
        help = "Encounter start command starts an encounter.",
        error = "encounter start [with: Target1 [and: Target2]*]",
        game = {
            steps = {
                {
                    name = "create_loop",
                    value = function()
                        return loop("encounter_start", true)
                    end,
                },
                {
                    name = "order_loop",
                    value = function()
                        return loop_order("encounter_start", false)
                    end,
                },
                {
                    name = "add_actors",
                    value = function()
                        return add_actor(command.with)
                    end,
                },
            },
        },
        targets = {
            steps = {
                {
                    name = "ask_initiative",
                    value = function()
                        return ask(target.id, "initiative")
                    end,
                },
            },
        },
    },

    encounter_end = {
        name = "encounter end",
        prereq = {
            {
                name = "check_conflict",
                value = function()
                    return is_encounter_start_active()
                end,
                error = "no active encounter to end",
            },
        },
        hint = "Encounter has ended.",
        help = "Encounter end command ends an encounter.",
        error = "encounter end",
        game = {
            steps = {
                {
                    name = "state_change",
                    value = function()
                        return loop("encounter_start", false)
                    end,
                },
            },
        },
    },

    encounter_add = {
        name = "encounter add",
        params = {
            { name = "with", type = "list<target>", required = true },
        },
        prereq = {
            {
                name = "check_conflict",
                value = function()
                    return is_encounter_start_active()
                end,
                error = "no active encounter to add to",
            },
        },
        hint = "Actor has been added to the encounter.",
        help = "Encounter add command adds an actor to the encounter.",
        error = "encounter add [with: Target1 [and: Target2]*]",
        game = {
            steps = {
                {
                    name = "add_actors",
                    value = function()
                        return add_actor(command.with)
                    end,
                },
            },
        },
    },

    initiative = {
        name = "initiative",
        prereq = {
            {
                name = "check_active",
                value = function()
                    return is_encounter_start_active()
                end,
                error = "an encounter is not active. Start it first",
            },
        },
        hint = "Is it your turn? Wait for your turn",
        help = "Initiative command rolls initiative for the actors.",
        error = "initiative",
        game = {
            steps = {
                {
                    name = "roll_score",
                    value = function()
                        return loop_value("encounter_start", roll("1d20") + mod(actor.stats.dex))
                    end,
                },
            },
        },
    },

    grapple = {
        name = "grapple",
        params = {
            { name = "to", type = "target", required = true },
        },
        prereq = {
            {
                name = "check_action",
                value = function()
                    return (actor.spent.actions or 0) < (actor.resources.actions or 0)
                end,
                error = "no actions remaining",
            },
        },
        hint = "Grapple command grapples the target.",
        help = "Grapple command grapples the target.",
        error = "grapple [to: <target>]",
        game = {
            steps = {
                {
                    name = "contest",
                    value = function()
                        local prof = 0
                        if actor.proficiencies.athletics then
                            prof = actor.proficiencies.athletics * (actor.stats.prof_bonus or 2)
                        end
                        return contest(roll("1d20") + mod(actor.stats.str) + prof)
                    end,
                },
            },
        },
        targets = {
            steps = {
                {
                    name = "ask_grapple",
                    value = function()
                        local val = game.contest.value or 10
                        return ask(
                            target.id,
                            "check skill: athletics dc: " .. tostring(val),
                            "check skill: acrobatics dc: " .. tostring(val)
                        )
                    end,
                },
                {
                    name = "resolve_grapple",
                    value = function()
                        return targets.ask_grapple
                    end,
                },
                {
                    name = "grappled",
                    value = function()
                        return condition("grappled")
                    end,
                },
            },
        },
        actor = {
            steps = {
                {
                    name = "consume_action",
                    value = function()
                        return spend("actions", 1)
                    end,
                },
            },
        },
    },

    move = {
        name = "move",
        params = {
            { name = "feet", type = "int", required = false },
            { name = "type", type = "string", required = false }, -- e.g. "speed", "fly", "swim"
        },
        prereq = {
            {
                name = "check_movement",
                value = function()
                    local mtype = command.type or "speed"
                    local mspeed = actor.resources[mtype] or 0
                    local spent = actor.spent[mtype] or 0
                    local feet = command.feet or mspeed -- Default to full speed if feet not provided
                    return feet <= (mspeed - spent)
                end,
                error = "not enough movement remaining",
            },
        },
        game = {
            steps = {
                {
                    name = "consume_move",
                    value = function()
                        local mtype = command.type or "speed"
                        local feet = command.feet or (actor.resources[mtype] or 0)
                        return spend(mtype, feet)
                    end,
                },
            },
        },
    },

    dash = {
        name = "dash",
        params = {
            { name = "type", type = "string", required = false },
        },
        prereq = {
            {
                name = "check_action",
                value = function()
                    return (actor.spent.actions or 0) < (actor.resources.actions or 0)
                end,
                error = "no actions remaining",
            },
            {
                name = "check_speed",
                value = function()
                    local mtype = command.type or "speed"
                    local mspeed = actor.resources[mtype] or 0
                    local spent = actor.spent[mtype] or 0
                    return mspeed == spent
                end,
                error = "spend all your movement before dashing",
            },
        },
        game = {
            steps = {
                {
                    name = "consume_action",
                    value = function()
                        return spend("actions", 1)
                    end,
                },
                {
                    name = "dash",
                    value = function()
                        local mtype = command.type or "speed"
                        return set_attr("spent", mtype, 0)
                    end,
                },
            },
        },
    },

    check = {
        name = "check",
        params = {
            { name = "skill", type = "string", required = true },
            { name = "dc", type = "int", required = true },
        },
        hint = "Check command checks the target.",
        help = "Check command checks the target.",
        error = "check [skill: <skill>] [dc: <dc>]",
        game = {
            steps = {
                {
                    name = "result",
                    value = function()
                        local ability = skill_to_ability(command.skill)
                        local prof = 0
                        if actor.proficiencies and actor.proficiencies[command.skill] then
                            prof = actor.proficiencies[command.skill] * (actor.stats.prof_bonus or 2)
                        end
                        local stat = 10
                        if actor.stats and actor.stats[ability] then
                            stat = actor.stats[ability]
                        end
                        return check_result((roll("1d20") + mod(stat) + prof) >= command.dc)
                    end,
                },
            },
        },
    },

    turn = {
        name = "turn",
        prereq = {
            {
                name = "check_active",
                value = function()
                    return is_encounter_start_active()
                end,
                error = "no active encounter",
            },
            {
                name = "check_actor_turn",
                value = function()
                    return current_actor().id == actor.id
                end,
                error = "not your turn",
            },
        },
        hint = "Next actor's turn.",
        help = "Ends the current actor's turn and advances to the next one in initiative order.",
        error = "end turn",
        game = {
            steps = {
                {
                    name = "advance",
                    value = function()
                        return next_turn("encounter_start")
                    end,
                },
            },
        },
    },

    disengage = {
        name = "disengage",
        prereq = {
            {
                name = "check_action",
                value = function()
                    return (actor.spent.actions or 0) < (actor.resources.actions or 0)
                end,
                error = "no actions remaining",
            },
        },
        hint = "Disengage command disengages the actor.",
        help = "Disengage command disengages the actor.",
        error = "disengage",
        game = {
            steps = {
                {
                    name = "consume_action",
                    value = function()
                        return spend("actions", 1)
                    end,
                },
                {
                    name = "disengage",
                    value = function()
                        return condition("disengaged")
                    end,
                },
            },
        },
        actor = {
            hooks = {
                {name = "end_disengage", type = "next_turn", value = function() return remove_condition("disengaged") end}
            }
        }
    },

    dodge = {
        name = "dodge",
        prereq = {
            {
                name = "check_action",
                value = function()
                    return (actor.spent.actions or 0) < (actor.resources.actions or 0)
                end,
                error = "no actions remaining",
            },
        },
        hint = "Dodge command lets the actor dodge incoming attacks.",
        help = "When you take the Dodge action, you focus entirely on avoiding attacks. Until the start of your next turn, any attack roll made against you has disadvantage if you can see the attacker, and you make Dexterity saving throws with advantage. You lose this benefit if you are incapacitated or if your speed drops to 0.",
        error = "dodge",
        game = {
            steps = {
                {
                    name = "consume_action",
                    value = function()
                        return spend("actions", 1)
                    end,
                },
                {
                    name = "dodge",
                    value = function()
                        return condition("dodging")
                    end,
                },
            },
        },
        actor = {
            hooks = {
                {name = "end_dodge", type = "next_actor_turn", value = function() return remove_condition("dodging") end}
            }
        }
    },

    add_condition = {
        name = "add condition",
        params = {
            { name = "condition", type = "string", required = true },
            { name = "to", type = "list<target>", required = true },
        },
        hint = "Condition added.",
        help = "Adds a condition to the specified targets. (GM only)",
        error = "add_condition [condition: <condition>] [to: Target1 [and: Target2]*]",
        targets = {
            steps = {
                {
                    name = "apply_condition",
                    value = function()
                        return condition(command.condition)
                    end,
                },
            },
        },
    },

    remove_condition = {
        name = "remove condition",
        params = {
            { name = "condition", type = "string", required = true },
            { name = "from", type = "list<target>", required = true },
        },
        hint = "Condition removed.",
        help = "Removes a condition from the specified targets. (GM only)",
        error = "remove_condition [condition: <condition>] [from: Target1 [and: Target2]*]",
        targets = {
            steps = {
                {
                    name = "do_remove",
                    value = function()
                        return remove_condition(command.condition)
                    end,
                },
            },
        },
    },

    hide = {
        name = "hide",
        prereq = {
            {
                name = "check_action",
                value = function()
                    return (actor.spent.actions or 0) < (actor.resources.actions or 0)
                end,
                error = "no actions remaining",
            },
        },
        hint = "Attempting to hide.",
        help = "With the Hide action, you try to conceal yourself. Make a Dexterity (Stealth) check. On success, you gain the Invisible condition.",
        error = "hide",
        game = {
            steps = {
                {
                    name = "stealth_check",
                    value = function()
                        local prof = 0
                        if actor.proficiencies and actor.proficiencies.stealth then
                            prof = actor.proficiencies.stealth * (actor.stats.prof_bonus or 2)
                        end
                        return roll("1d20") + mod(actor.stats.dex) + prof
                    end,
                },
                {
                    name = "notify_result",
                    value = function()
                        if game.stealth_check >= 15 then
                            return hint("Stealth check total: " .. tostring(game.stealth_check) .. " (Success - Use as DC to find)")
                        else
                            return hint("Stealth check total: " .. tostring(game.stealth_check) .. " (Failure)")
                        end
                    end,
                },
            },
        },
        actor = {
            steps = {
                {
                    name = "consume_action",
                    value = function()
                        return spend("actions", 1)
                    end,
                },
                {
                    name = "evaluate_hide",
                    value = function()
                        if game.stealth_check >= 15 then
                            return condition("invisible")
                        end
                        return nil
                    end,
                },
            },
        },
    },

    improvise = {
        name = "improvise",
        prereq = {
            {
                name = "check_action",
                value = function()
                    return (actor.spent.actions or 0) < (actor.resources.actions or 0)
                end,
                error = "no actions remaining",
            },
        },
        hint = "Awaiting GM adjudication for improvised action.",
        help = "When you describe an action not detailed elsewhere in the rules, the Dungeon Master tells you whether that action is possible and what kind of D20 Test you need to make.",
        error = "improvise",
        actor = {
            steps = {
                {
                    name = "consume_action",
                    value = function()
                        return spend("actions", 1)
                    end,
                },
            },
        },
    },
}
