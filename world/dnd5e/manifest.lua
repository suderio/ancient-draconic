restrictions = {
  adjudication = {
    commands = {"grapple"}
  },
  gm_commands = {"encounter_start", "encounter_end"}
}

local _sizes_list = {'tiny', 'small', 'medium', 'large', 'huge', 'gargantuan'}
local _skill_map = { athletics = 'str', acrobatics = 'dex', stealth = 'dex', sleight_of_hand = 'dex', perception = 'wis', insight = 'wis', survival = 'wis', investigation = 'int', arcana = 'int', history = 'int', religion = 'int', nature = 'int', medicine = 'wis', persuasion = 'cha', intimidation = 'cha', performance = 'cha', deception = 'cha' }

function mod(val)
    if not val then return -5 end
    return math.floor(val / 2) - 5
end

function sizes(input)
    for i, name in ipairs(_sizes_list) do
        if name == input then return i end
    end
    return 0
end

function skill_to_ability(skill)
    return _skill_map[skill] or 'str'
end

commands = {
  encounter_start = {
    name = "encounter start",
    params = {
      { name = "with", type = "list<target>", required = false }
    },
    prereq = {
      { 
        name = "check_conflict", 
        formula = function() return not is_encounter_start_active end, 
        error = "an encounter is already active. End it first" 
      }
    },
    hint = "Encounter has started. Roll initiative for all actors.",
    help = "Encounter start command starts an encounter.",
    error = "encounter start [with: Target1 [and: Target2]*]",
    game = {
      { name = "create_loop", formula = true, event = "LoopEvent" },
      { name = "order_loop", formula = false, event = "LoopOrderAscendingEvent" },
      { name = "add_actor", formula = function() return command.with end, event = "ActorAddedEvent" }
    },
    targets = {
      -- Send initiative request to all participants
      { name = "Ask Initiative", formula = function() return {target.id, "initiative"} end, event = "AskIssuedEvent" }
    }
  },

  encounter_end = {
    name = "encounter end",
    prereq = {
      { 
        name = "check_conflict", 
        formula = function() return is_encounter_start_active end, 
        error = "no active encounter to end" 
      }
    },
    hint = "Encounter has ended.",
    help = "Encounter end command ends an encounter.",
    error = "encounter end",
    game = {
      { 
        name = "state_change", 
        formula = false, 
        loop = "encounter_start", 
        event = "LoopEvent" 
      }
    }
  },

  initiative = {
    name = "initiative",
    prereq = {
      { 
        name = "check_active", 
        formula = function() return is_encounter_start_active end, 
        error = "an encounter is not active. Start it first" 
      }
    },
    hint = "Is it your turn? Wait for your turn",
    help = "Initiative command rolls initiative for the actors.",
    error = "initiative",
    game = {
      { 
        name = "roll_score", 
        formula = function() return roll("1d20") + mod(actor.stats.dex) end, 
        event = "LoopOrderEvent",
        loop = "encounter_start"
      }
    }
  },

  grapple = {
    name = "grapple",
    params = {
      { name = "to", type = "target", required = true }
    },
    prereq = {
      { 
        name = "check_action", 
        formula = function() return actor.spent.actions < actor.resources.actions end, 
        error = "no actions remaining" 
      }
    },
    hint = "Grapple command grapples the target.",
    help = "Grapple command grapples the target.",
    error = "grapple [to: <target>]",
    game = {
      { 
        name = "contest", 
        formula = function() 
            local prof = 0
            if actor.proficiencies.athletics then 
                prof = actor.proficiencies.athletics * (actor.stats.prof_bonus or 2)
            end
            return roll("1d20") + mod(actor.stats.str) + prof
        end, 
        event = "ContestStarted" 
      }
    },
    targets = {
      { 
        name = "ask_grapple", 
        formula = function() 
            -- The ContestStarted event writes into game state metadata via the Go executor,
            -- but the executor also returned it in `game.contest`.
            -- Let's extract exactly the value. The executor sets it to a map: map[string]any{"actor": ..., "value": 15}
            local val = game.contest.value or 10
            return {target.id, "check skill: athletics dc: " .. tostring(val), "check skill: acrobatics dc: " .. tostring(val)}
        end,
        event = "AskIssuedEvent" 
      },
      { 
        name = "resolve_grapple", 
        formula = function() return targets.ask_grapple end, 
        event = "ContestResolvedEvent" 
      },
      { 
        name = "grappled", 
        formula = "grappled", 
        event = "AddConditionEvent" 
      }
    },
    actor = {
      { name = "consume_action", formula = "actions", event = "AddSpentEvent" }
    }
  },

  check = {
    name = "check",
    params = {
      { name = "skill", type = "string", required = true },
      { name = "dc", type = "int", required = true }
    },
    hint = "Check command checks the target.",
    help = "Check command checks the target.",
    error = "check [skill: <skill>] [dc: <dc>]",
    game = {
      { 
        name = "contest", 
        formula = function()
            local ability = skill_to_ability(command.skill)
            local prof = 0
            if actor.proficiencies and actor.proficiencies[command.skill] then 
                prof = actor.proficiencies[command.skill] * (actor.stats.prof_bonus or 2) 
            end
            local stat = 10
            if actor.stats and actor.stats[ability] then
                stat = actor.stats[ability]
            end
            return (roll("1d20") + mod(stat) + prof) >= command.dc
        end, 
        event = "CheckEvent" 
      }
    }
  }
}
