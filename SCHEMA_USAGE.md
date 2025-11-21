# Using the JSON Schema for VS Code Autocomplete

The `generate-schema` command creates a JSON Schema file that enables VS Code to provide intelligent autocomplete for your gig YAML files.

## Generating the Schema

```bash
# Generate schema from config.yaml (default)
gigsheets generate-schema

# Generate schema from a specific config file
gigsheets generate-schema --config path/to/config.yaml --output path/to/schema.json
```

## Setting up VS Code

After generating the schema, add it to your VS Code settings to enable autocomplete:

1. Open VS Code settings (Ctrl/Cmd + ,)
2. Search for "yaml.schemas"
3. Click "Edit in settings.json"
4. Add the schema mapping:

```json
{
  "yaml.schemas": {
    "./gig-schema.json": "*.yaml"
  }
}
```

Or for a specific pattern:

```json
{
  "yaml.schemas": {
    "./gig-schema.json": "**/gig*.yaml"
  }
}
```

## What You Get

The schema provides autocomplete for:

- **Song nicknames**: All songs defined in your config.yaml
- **Image variants**: For songs with multiple images, you'll get completions like:
  - `song1` (default image)
  - `song2#v2` (variant image)
  - `song2#alternate` (another variant)

## Example

Given a config.yaml with:

```yaml
songs:
  - nickname: wonderful-tonight
    images:
      default: images/wonderful-tonight.png
      simplified: images/wonderful-tonight-simple.png
  - nickname: layla
    image: images/layla.png
```

When editing a gig.yaml file, you'll get autocomplete suggestions for:
- `wonderful-tonight`
- `wonderful-tonight#simplified` 
- `layla`

## Updating the Schema

Whenever you add or modify songs in your config.yaml, regenerate the schema to update the autocomplete suggestions:

```bash
gigsheets generate-schema
```