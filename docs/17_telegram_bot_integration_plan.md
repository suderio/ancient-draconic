# Phase 17: Telegram Bot Integration

Enable remote play via Telegram by integrating a polling bot that maps chat users to character actors in the DSL.

## Clarifications & Final Decisions

- **Execution Mode**: The bot runner is integrated into the REPL/TUI startup. If a campaign has telegram configured, launching the REPL spawns a background worker for polling.
- **Message Formatting**: Responses are converted to Markdown for Telegram compatibility.
- **Single Campaign Lock**: Only one campaign session can be active at a time. Updates from unexpected `chat_id`s are ignored.
- **Command Filtering**: Only messages starting with `/` are processed. `/command` maps to `command by: <actor>`.
- **Offset Persistence**: `last_update_id` is saved globally to ensure no message loss between restarts.

## Proposed Changes

### 1. Global Configuration (`bot telegram`)

- **[NEW] `cmd/bot.go`**: Implement `draconic bot telegram`.
 - Interactive setup with `@BotFather` instructions.
 - Save token to global `.draconic.yaml` via Viper.

### 2. Campaign-Specific Configuration (`campaign telegram`)

- **[MODIFY] `cmd/campaign.go`**: Add `telegram` sub-command.
 - Save `chat_id` and user mappings to `telegram.yaml` within the campaign directory.
 - Validate mapped usernames against character/monster data.

### 3. Telegram Engine (`internal/telegram`)

- **[NEW] `client.go`**: HTTP client for `getUpdates` (long polling) and `sendMessage`.
- **[NEW] `worker.go`**: Background worker loop for command translation and event formatting.

## Verification Plan

- Unit tests for command translation and chat filtering using API mocks.
- End-to-end verification with a live test bot.
