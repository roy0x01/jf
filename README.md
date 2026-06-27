<div align="center">

<img src="logo.svg" width="120" height="120" alt="jf logo" />

# jf

**JSON formatter and filter for daily terminal use**

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-EAB308?style=flat-square)
![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macOS%20%7C%20windows-6B7280?style=flat-square)

</div>

---

`jf` pretty-prints JSON and JSONL files with syntax highlighting and lets you filter records using a simple deep path syntax — no jq required.

## Install

```bash
go install github.com/roy0x01/jf@latest
```

**Binary install:**

```bash
# Linux (amd64)
curl -L https://github.com/roy0x01/jf/releases/latest/download/jf-linux-amd64 -o jf
chmod +x jf && sudo mv jf /usr/local/bin/

# Linux (arm64)
curl -L https://github.com/roy0x01/jf/releases/latest/download/jf-linux-arm64 -o jf
chmod +x jf && sudo mv jf /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/roy0x01/jf/releases/latest/download/jf-darwin-amd64 -o jf
chmod +x jf && sudo mv jf /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/roy0x01/jf/releases/latest/download/jf-darwin-arm64 -o jf
chmod +x jf && sudo mv jf /usr/local/bin/

# Windows (amd64)
curl -L https://github.com/roy0x01/jf/releases/latest/download/jf-windows-amd64.exe -o jf.exe
```

---

## Usage

```
jf <input> [-o output] [--filter path[=value]]
```

| Command | What it does |
|---|---|
| `jf data.json` | Pretty-print with color |
| `jf data.json -o pretty.json` | Pretty-print and save to file |
| `jf logs.jsonl --filter key=value` | Filter matching records |
| `jf logs.jsonl --filter key=value -o out.jsonl` | Filter and save |

The input file is **never modified** unless `-o` points back to it.

---

## Color scheme

| Token | Color |
|---|---|
| Keys | Sky blue |
| Strings | Sage green |
| Numbers | Coral |
| Booleans | Violet |
| Null | Grey |
| `{ } [ ]` | Gold |

---

## Filter

Filter with a dot-notation path, optionally with a value to match.

```bash
# key exists
jf logs.jsonl --filter event

# key equals value
jf logs.jsonl --filter event=login
jf logs.jsonl --filter member.role=admin
jf logs.jsonl --filter member.active=false

# array wildcard [*]
jf scan.json --filter hosts[*].ip=10.0.0.1
jf scan.json --filter hosts[*].ports[*].state=open

# specific array index
jf data.jsonl --filter results.0.severity=critical

# filter and save
jf audit.jsonl --filter severity=critical -o hits.jsonl
```

### Path syntax

| Expression | Meaning |
|---|---|
| `key` | Top-level key |
| `a.b` | Nested key |
| `a.0.b` | Array index then key |
| `a[*].b` | All array elements, then key |
| `a[*].b[*].c` | Nested wildcards |

### Filter behaviour by file type

| Input | Behaviour |
|---|---|
| JSONL | Each line is a record — matching lines are kept |
| JSON array `[…]` | Each element tested independently |
| JSON object `{…}` | Whole document tested as one record |

---

## License

MIT
