# Test: Full sync workflow
# Creates a registry + project + profile, then syncs and verifies output.

Output e2e/actual/sync_workflow.gif

Set Shell "bash"
Set Width 100
Set Height 30

# Setup isolated environment
Type "export HYSTAK_CONFIG_DIR=$(mktemp -d)"
Enter
Type "export PROJECT_DIR=$(mktemp -d)"
Enter
Sleep 500ms

Type@50ms "mkdir -p $HYSTAK_CONFIG_DIR/profiles $PROJECT_DIR/.git"
Enter
Sleep 500ms

# Create registry with a server
Type@50ms "cat > $HYSTAK_CONFIG_DIR/registry.yaml << 'YAML'"
Enter
Type@50ms "mcps:"
Enter
Type@50ms "  github:"
Enter
Type@50ms "    transport: stdio"
Enter
Type@50ms "    command: npx"
Enter
Type@50ms "    args: [\"-y\", \"@anthropic/mcp-github\"]"
Enter
Type@50ms "    env:"
Enter
Type@50ms "      GITHUB_TOKEN: \"${GITHUB_TOKEN}\""
Enter
Type@50ms "skills: {}"
Enter
Type@50ms "hooks: {}"
Enter
Type@50ms "permissions: {}"
Enter
Type@50ms "templates: {}"
Enter
Type@50ms "prompts: {}"
Enter
Type@50ms "tags: {}"
Enter
Type@50ms "YAML"
Enter
Sleep 500ms

# Create project
Type@50ms "cat > $HYSTAK_CONFIG_DIR/projects.yaml << YAML"
Enter
Type@50ms "projects:"
Enter
Type@50ms "  myproject:"
Enter
Type@50ms "    path: $PROJECT_DIR"
Enter
Type@50ms "    active_profile: dev"
Enter
Type@50ms "    managed_mcps: []"
Enter
Type@50ms "YAML"
Enter
Sleep 500ms

# Create profile
Type@50ms "cat > $HYSTAK_CONFIG_DIR/profiles/dev.yaml << 'YAML'"
Enter
Type@50ms "name: dev"
Enter
Type@50ms "mcps:"
Enter
Type@50ms "  - github"
Enter
Type@50ms "YAML"
Enter
Sleep 500ms

# Run sync
Type "hystak sync myproject"
Enter
Sleep 1s

# Check the deployed config
Type "cat $PROJECT_DIR/.mcp.json"
Enter
Sleep 1s

# Run diff (should show no drift)
Type "hystak diff myproject"
Enter
Sleep 1s

# Clean up
Type "rm -rf $HYSTAK_CONFIG_DIR $PROJECT_DIR"
Enter
Sleep 500ms
