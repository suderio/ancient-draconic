# Operations: File Handling, Config, and Secrets

## Purpose

Define filesystem, configuration, and secret-handling rules for resource persistence and safe operation.

## Normative Rules

### Filesystem Safety

1. All paths MUST be normalized before IO.
2. Filesystem joins MUST reject traversal outside configured roots.
3. Save operations MUST be atomic at file level.
4. Filesystem operations MUST be idempotent for repeated equivalent inputs.

### Configuration and Secrets

1. Secret values (API tokens, credentials) MUST be stored in Viper config files, never in source or campaign data.
2. Secret values MUST never appear in logs, error messages, or event logs.
3. The Telegram bot token is resolved from `telegram_token` in the Viper config.
4. Campaign-specific Telegram config (`telegram.yaml`) stores chat ID and user mappings (non-secret data).

## Data Contracts

Campaign directory layout:

```shell
worlds/<world>/
  manifest.lua         ← game rules (Lua)
  manifest.yaml        ← legacy fallback
  data/
    characters/*.yaml  ← entity data (YAML, future: .lua)
    monsters/*.yaml
  <campaign>/
    log.jsonl          ← append-only event log
    characters/*.yaml  ← campaign-specific entities
    telegram.yaml      ← optional bot config
```

## Failure Modes

1. Path traversal outside configured roots.
2. Secret values leaked in error output or logs.
3. Non-atomic writes corrupting the event log.
