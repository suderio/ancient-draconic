# Hierarchy Data Resolver Plan

## Objective

The user wants to allow the system to load `data/` files from a hierarchy of directories to enable isolated campaigns/worlds to possess custom monsters or character sheets.
Additionally, the user requested that we unify search logic inside `loader.go` and remove code duplication in `CheckEntityLocally`.

## Directory Hierarchy

The engine will attempt to load YAML data definitions in this exact order:

1. `<campaign_path>/data` : Campaign-specific overrides
2. `<world_path>/data` : World-specific overrides
3. `./data` (Root dir) : System default SRD records

## Refactoring Steps

1. **Update `internal/data/loader.go`:**
  - Modify `NewLoader` to accept `dataDirs []string` instead of a single `dataDir string`.
  - Update `load(ref string, target interface{})` to iterate over the `dataDirs` array, returning the first file it can successfully open and unmarshal.
  - Implement `LoadCharacter(name string) (*Character, error)` and update `LoadMonster` to construct proper hierarchical filenames seamlessly.

2. **Update `internal/command/utils.go`:**
  - Modify `CheckEntityLocally` to accept `loader *data.Loader` instead of `baseDir string`.
  - Update `CheckEntityLocally` logic to use `loader.LoadCharacter` and `loader.LoadMonster` directly, cutting out all of the `filepath.Join` boilerplate that exists there currently.

3. **Update Command Hooks (`add.go`, `encounter.go`, `initiative.go`):**
  - Propagate the updated `loader *data.Loader` argument signature downstream from the `Session` router.

4. **Update `internal/session/session.go`:**
  - Update `Session` struct to hold `loader *data.Loader` instead of `baseDir string`.
  - Update `NewSession` to accept `dataDirs []string`, build the Loader object natively, and assign it to the Session memory.

5. **Update `cmd/repl.go`:**
  - In the `replCmd`, compute the `[]string` array holding: `campaign/data`, `world/data`, `root/data`.
  - Pass this array into `NewSession(dataDirs, store)`.

6. **Verify** functionality via tests to prove system checks custom characters seamlessly before standard ones.
