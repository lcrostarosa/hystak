# Claude Code Configuration Formats

## Discovery Targets

| File | Scope | Contents |
|------|-------|----------|
| `~/.claude.json` | Global | MCPs (`mcpServers` key) + Claude prefs |
| `.mcp.json` | Project | MCPs (`mcpServers` key) |
| `~/.claude/settings.json` | Global | Hooks, permissions, env vars |
| `.claude/settings.local.json` | Project | Hooks, permissions, env vars |
| `~/.claude/skills/*/SKILL.md` | Global | Skill markdown files |
| `.claude/skills/*/SKILL.md` | Project | Skill markdown files |
| `CLAUDE.md` | Project | Project instructions |
| `~/.claude/CLAUDE.md` | Global | Global instructions |

## .mcp.json Schema

```json
{
  "mcpServers": {
    "server-name": {
      "type": "stdio" | "http" | "sse",
      "command": "string",
      "args": ["array"],
      "env": { "KEY": "value" },
      "url": "https://...",
      "headers": { "Key": "value" }
    }
  }
}
```

## settings.local.json Schema

```json
{
  "env": { "KEY": "value" },
  "hooks": {
    "EventType": [
      {
        "matcher": "optional-regex",
        "hooks": [{ "type": "command", "command": "...", "timeout": 15000 }]
      }
    ]
  },
  "permissions": {
    "allow": ["Bash(*)", "WebFetch(domain:github.com)"],
    "deny": ["Bash(rm:*)"]
  }
}
```

Hook events: UserPromptSubmit, Stop, PreToolUse, PostToolUse

## Skills Structure

```
.claude/skills/
├── skill-name/
│   └── SKILL.md
└── .hystak-managed    (one name per line)
```

## CLAUDE.md Management

- Managed files start with `<!-- managed by hystak -->`
- Files without sentinel are user-owned and never overwritten

## Env Var Syntax

Preserved as-is: `${VAR}` and `${env:VAR}` both valid.
