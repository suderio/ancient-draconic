# Persistence

To handle multiple sessions and long-term campaigns in an event-sourced engine, we don't save the "character sheets"—we save the **Event Log** itself. Since your engine already treats every action as a permanent fact in a sequence, "Persistence" simply means writing that sequence to a file or database.

### 1. The World vs. Campaign vs. Session Hierarchy

We can structure the persistence layer to distinguish between the overarching world state and specific play sessions:

* **World:** This is the root directory of the entire "World". Here we can add new yaml files that will be shared between campaigns.
* **Campaign:** This contains the **Long-Term Log** (Level-ups, inventory changes, world events).
* **Session:** A subset of events or a "Save Point" within a campaign. A session is not created in a different directory, just marked as an event in the log.
* **Active Encounter:** A temporary, high-frequency log that can be merged into the Campaign log once combat ends.

### 2. Implementation: The File-Based Event Store

Since we are using **Viper** and a local directory for SRD data, we can use a similar approach for saves. I recommend a **JSONL (JSON Lines)** format for the event log.

* **Append-Only Performance:** Every time a command is processed, the engine appends one line of JSON to the `campaign_log.jsonl` file.
* **Human Readable:** Like the YAML SRD files, a user can open the log to see exactly what happened in their game.
* **Crash Recovery:** If the program crashes, the engine just re-reads the file from the start to rebuild the state.

### 3. The `Campaign` Manager

The engine needs a way to "Load" these logs. In our **Cobra CLI**, this would involve a new set of commands:

| Command | Action |
| --- | --- |
| `dnd campaign create "The Lost Mine"` | Creates a new directory The-Lost-Mine and an empty `log.jsonl`. |
| `dnd campaign load "The Lost Mine"` | Replays all events to populate the `GameState`. |
| `dnd session save` | (Optional) Creates a "Snapshot" event in the log to speed up future loading. TBD. |

### 4. Persistence & Snapshots

As the campaign grows to thousands of events, replaying from turn 1 can get slow.

* **Snapshots:** Every 100 events, or by session, the engine can serialize the entire `GameState` (including the HP of characters, number of Hit Dice used, etc.) into a `snapshot.yaml` file.
* **Loading:** The engine loads the latest snapshot first, then only replays the events that happened *after* that snapshot.

### 5. Mission Brief: Persistence Layer

> **Mission: Implementation of Campaign Persistence**
> 1. **Viper Configuration:** Add `worlds_dir` setting to the config. Default to `./worlds`
> 2. **JSONL Serializer:** Create a package `internal/persistence` that can encode/decode your `Event` interface to JSON.
> 3. **Directory Structure:** Each new world and campaign created under the `worlds_dir` follows the same structure as the `data_dir` after the execution of `init`. They will have the same subdirectories. They will also have one more directory called `characters`.
> 4. **Integration:** Modify the `repl` command so that every successful `CommandResponse` triggers a `persistence.Append(event)` call.
> 5. **Validation:** Test that after closing and reopening the CLI, a **Zombie** wounded in a previous session still has its reduced HP.
