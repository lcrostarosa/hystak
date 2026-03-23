# Test: hystak doctor validates registry
# Creates a registry and runs doctor to check validity.

Output e2e/actual/doctor.gif

Set Shell "bash"
Set Width 100
Set Height 25

# Create isolated config
Type "export HYSTAK_CONFIG_DIR=$(mktemp -d)"
Enter
Sleep 500ms

Type@50ms "mkdir -p $HYSTAK_CONFIG_DIR/profiles"
Enter
Sleep 500ms

# Create a clean registry
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

# Create empty projects
Type@50ms "cat > $HYSTAK_CONFIG_DIR/projects.yaml << 'YAML'"
Enter
Type@50ms "projects: {}"
Enter
Type@50ms "YAML"
Enter
Sleep 500ms

# Run doctor
Type "hystak doctor"
Enter
Sleep 1s

# Clean up
Type "rm -rf $HYSTAK_CONFIG_DIR"
Enter
Sleep 500ms
