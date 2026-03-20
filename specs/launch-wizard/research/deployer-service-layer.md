# Deployer & Service Layer Research

## Deployer Interface

```go
type Deployer interface {
    ClientType() model.ClientType
    ConfigPath(projectPath string) string
    ReadServers(projectPath string) (map[string]model.ServerDef, error)
    WriteServers(projectPath string, servers map[string]model.ServerDef) error
    Bootstrap(projectPath string) error
}
```

Plus specialized deployers: SkillsDeployer, SettingsDeployer, ClaudeMDDeployer.

## Sync Flow

```
SyncProject(projectName)
  ├─ Resolve servers (tags + direct MCPs, apply overrides)
  ├─ For each client:
  │   ├─ Bootstrap → ReadServers → Backup → Merge (unmanaged + expected) → WriteServers
  │   └─ Track results (added/updated/unchanged/unmanaged)
  ├─ SyncSkills → deploy to .claude/skills/, track via .hystak-managed
  ├─ SyncSettings → write hooks/permissions to settings.local.json
  └─ SyncClaudeMD → write template to CLAUDE.md with sentinel
```

## Changes Needed for Profiles

- `Project` model needs `Profiles map[string]ProfileDef` sub-entity
- `ResolveServers()` needs profile parameter
- `SyncProject()` needs profile selection
- projects.yaml format changes to nest profile data

## Changes Needed for Symlink Deploys

- Skills: replace `.hystak-managed` marker with symlinks (symlink IS the marker)
- CLAUDE.md: replace `<!-- managed by hystak -->` sentinel with symlink
- `.mcp.json`: can't symlink (single JSON file), keep managed/unmanaged tracking
- Settings: can't symlink (single JSON file), keep key-level tracking

## Service Public API (key methods)

- Sync: SyncProject, SyncAll, PreflightSync, DriftReport, Diff
- CRUD: Add/Update/Delete/List/Get for Server, Skill, Hook, Permission, Template
- Project: Add/Delete/List/Get + Assign/Unassign + SetOverride
- Import: ImportFromFile, ApplyImport
- Backup: BackupConfigs, ListBackups, RestoreBackup
- Discovery: ScanForConfigs, IsEmpty
