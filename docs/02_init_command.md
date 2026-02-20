# Init Command

## **1. Objective**

Implement the `init` command to bootstrap the local game data environment. This command will fetch the 5e SRD data, transform it, and store it locally for offline use and user modification.

## **2. Technical Requirements**

* **Configuration (Viper):** * Load settings from a `.dndsl.yaml` file.
* Configurable `data_dir` (default: `./data`).

* **Data Pipeline:**
* Fetch core indices (Spells, Monsters, Classes, Features) from `https://www.dnd5eapi.co/api/`.
* Iterate through each item and fetch the full detail JSON.

* **Transformation:**
* **JSON  YAML:** Convert the payload to a clean YAML format.
* **Link Localization:** Regex or string-replace API URLs (e.g., `/api/2014/spells/vicious-mockery`) with relative local paths (e.g., `@ref:spells/vicious-mockery.yaml`). I.e. the ref property must show the relative local path to the file (under the `./data` directory)

* **CLI UX:**
* Use a progress bar (e.g., `github.com/schollz/progressbar`) to show download status.

## **3. Implementation Guardrails**

* **Rate Limiting:** Implement a slight delay between API calls to be a good citizen to the dnd5eapi servers.
* **Idempotency:** If a file already exists, the `init` command should have a `--force` flag to overwrite it; otherwise, skip to save bandwidth.
* **restrict apis downloaded:** Create a flag for every api so that we can update only some of them, like: `dnd init --alignments --races --force`. If no flag is given, updates all.

## **4. Proposed Changes**

### Dependencies

* `github.com/spf13/viper`: For configuration (data directory).
* `github.com/schollz/progressbar/v3`: For download progress.
* `gopkg.in/yaml.v3`: For YAML serialization.

### API Client (`pkg/dnd5eapi`)

* Implement `Client` to fetch data from the API.
* Implement `DownloadAll` to iterate over 24 endpoints: "spells", "monsters", "classes", "ability-scores", "alignments", "backgrounds", "conditions", "damage-types", "equipment", "equipment-categories", "feats", "features", "languages", "magic-items", "magic-schools", "proficiencies", "races", "rule-sections", "rules", "skills", "subclasses", "subraces", "traits", "weapon-properties"

* Implement Transformation Logic:
  * Convert JSON response to YAML.
  * Rewrite API URLs (`/api/2014/classes/wizard`) to relative local references (`classes/wizard.yaml`).
* Enhance the `init` command to automatically download images referenced in `monsters`, `equipment`, and `magic-items` endpoints. The `image` property in the generated YAML should be updated to point to the local file.
  * Iterate through keys to find "image".
  * If found, extract the URL path (e.g., `/api/images/weapon.png`).
  * Construct full download URL.
  * Construct local file path (`data/magic-items/weapon.png`).
  * Download the file.
  * Update the map value to the new local path.
* Handle `resource_list_url` field by transforming it to `resource_list_ref` in a similar way to the url -> ref transformation.
* In the subclasses/*.yaml files, handle the subclass_levels property  by creating a new folder called levels under the subclasses folder and downloading the levels to it.
  * Example: /api/2014/subclasses/evocation/levels -> ./data/subclasses/evocation/levels.yaml
* Do the same with the class_levels property in the classes/*.yaml files.
  * Example: /api/2014/classes/wizard/levels -> ./data/classes/wizard/levels.yaml
* In the features/*.yaml files handle the spell, feature and reference properties in the same way:
  * Example: spell: /api/2014/spells/eldritch-blast -> spell: spells/eldritch-blast.yaml  
  * Example: feature: /api/2014/features/action-surge -> feature: features/action-surge.yaml  
  * Example: reference: /api/2014/rules/action-surge -> reference: rules/action-surge.yaml
  * Example: reference: /api/2014/classes/cleric/spellcasting -> reference: classes/cleric/spellcasting.yaml  
* Treat the resource_list_url property in background/*.yaml file the same way as the url property in the other files.

### CLI (`cmd/init.go`)

* Create `init` command.
* Load `data_dir` from Viper config (default `./data`).
* Trigger download process.

### Verification Plan

* Run `dnd init` and verify `data/` directory structure.
* Check generated YAML files for correct format and references.
* **Manual Test**:
  * Run `dnd init --equipment --force` (since equipment often has images).
  * Check if `.png` or `.jpg` files appear in `data/equipment/`.
  * Check if `data/equipment/....yaml` has updated `image` field.
