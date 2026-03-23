# Test: hystak list shows registered servers
# Creates a registry with servers, then lists them.

Output e2e/actual/list_servers.gif

Set Shell "bash"
Set Width 100
Set Height 20

# Create a config dir for this test
Type "export HYSTAK_CONFIG_DIR=$(mktemp -d)"
Enter
Sleep 500ms

# Create a minimal registry with servers
Type@50ms "mkdir -p $HYSTAK_CONFIG_DIR"
Enter
Sleep 500ms

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
Type@50ms "  postgres:"
Enter
Type@50ms "    transport: stdio"
Enter
Type@50ms "    command: npx"
Enter
Type@50ms "    args: [\"-y\", \"@anthropic/mcp-postgres\"]"
Enter
Type@50ms "YAML"
Enter
Sleep 500ms

# List servers
Type "hystak list"
Enter
Sleep 1s

# Clean up
Type "rm -rf $HYSTAK_CONFIG_DIR"
Enter
Sleep 500ms
