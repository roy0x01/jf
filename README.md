<div align="center">

<svg width="80" height="80" viewBox="0 0 80 80" xmlns="http://www.w3.org/2000/svg">
  <rect width="80" height="80" rx="16" fill="#0d1117"/>
  <text x="12" y="52" font-family="monospace" font-size="38" font-weight="bold" fill="#EAB308">{</text>
  <text x="44" y="52" font-family="monospace" font-size="38" font-weight="bold" fill="#75B8F0">}</text>
  <rect x="10" y="58" width="60" height="3" rx="1.5" fill="#EAB308" opacity="0.4"/>
  <text x="22" y="73" font-family="monospace" font-size="10" fill="#6B7280" letter-spacing="6">jf</text>
</svg>

# jf

**JSON formatter and filter for daily use**

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-EAB308?style=flat-square)
![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macOS%20%7C%20windows-6B7280?style=flat-square)

</div>

---

## Install

**With Go:**
```bash
go install github.com/roy0x01/jf@latest
```

**Binary download** — [releases page](https://github.com/roy0x01/jf/releases/latest)

```bash
# Linux
curl -L https://github.com/roy0x01/jf/releases/latest/download/jf-linux-amd64 -o jf
chmod +x jf && sudo mv jf /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/roy0x01/jf/releases/latest/download/jf-darwin-arm64 -o jf
chmod +x jf && sudo mv jf /usr/local/bin/
```

---

## Usage

```
jf <input> [-o output] [--filter path[=value]]
```

| Command | Description |
|---|---|
| `jf data.json` | Pretty-print with color to terminal |
| `jf data.json -o pretty.json` | Save plain output to file |
| `jf logs.jsonl --filter key=value` | Filter and print matching records |
| `jf logs.jsonl --filter key=value -o out.jsonl` | Filter and save |

---

## Features

**Auto-detects JSON and JSONL** — no flags needed, format is inferred from content.

**Syntax highlighting**

| Token | Color |
|---|---|
| Keys | Sky blue |
| Strings | Sage green |
| Numbers | Coral |
| Booleans | Violet |
| Null | Grey |
| Braces / Brackets | Gold |

**Deep path filtering** — traverse nested objects and arrays with a simple path syntax.

**Files are never modified** unless `-o` is specified.

---

## Filter path syntax

```
key                   top-level key
a.b                   nested key
a.0.b                 array index then key
a[*].b                wildcard — all array elements, then key b
a[*].b[*].c           nested wildcards
```

### Examples

```bash
# existence check — records that have the key
jf logs.jsonl --filter event

# value match
jf logs.jsonl --filter event=login
jf logs.jsonl --filter member.role=admin
jf logs.jsonl --filter member.active=false

# array wildcard
jf scan.json --filter hosts[*].ip=10.0.0.1
jf scan.json --filter hosts[*].ports[*].state=open

# nested wildcards
jf dump.json --filter teams[*].members[*].role=lead

# specific index
jf data.jsonl --filter results.0.severity=critical

# filter + save
jf audit.jsonl --filter severity=critical -o hits.jsonl
```

---

## Filter behaviour

| Input type | Filter behaviour |
|---|---|
| JSONL | Each line is a record; matching lines are kept |
| JSON array `[…]` | Each element tested independently; matching elements returned |
| JSON object `{…}` | Whole document tested as one record |

---

## Sample files

[`sample.json`](sample.json) — nested object with teams, members, ops, and meta  
[`sample.jsonl`](sample.jsonl) — 5 op records with nested member objects

Try it:
```bash
jf sample.json
jf sample.jsonl --filter severity=critical
jf sample.json --filter "teams[*].members[*].role=lead"
```

---

## License

MIT
