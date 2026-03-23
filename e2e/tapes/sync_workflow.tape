Output e2e/actual/sync_workflow.gif

Set Shell "bash"
Set Width 120
Set Height 120

Type "export HYSTAK_CONFIG_DIR=$(mktemp -d)"
Enter
Type "export PROJECT_DIR=$(mktemp -d)"
Enter
Sleep 500ms

Type "mkdir -p $HYSTAK_CONFIG_DIR/profiles $PROJECT_DIR/.git"
Enter
Sleep 500ms

# Create minimal registry
Type "cat > $HYSTAK_CONFIG_DIR/registry.yaml << 'EOF'"
Enter
Type "mcps:"
Enter
Type "  github:"
Enter
Type "    transport: stdio"
Enter
Type "    command: npx"
Enter
Type "skills: {}"
Enter
Type "hooks: {}"
Enter
Type "permissions: {}"
Enter
Type "templates: {}"
Enter
Type "prompts: {}"
Enter
Type "tags: {}"
Enter
Type "EOF"
Enter
Sleep 500ms

# Create project
Type "printf 'projects:\n  myproject:\n    path: '\"$PROJECT_DIR\"'\n    active_profile: dev\n    managed_mcps: []\n' > $HYSTAK_CONFIG_DIR/projects.yaml"
Enter
Sleep 500ms

# Create profile
Type "cat > $HYSTAK_CONFIG_DIR/profiles/dev.yaml << 'EOF'"
Enter
Type "name: dev"
Enter
Type "mcps:"
Enter
Type "  - github"
Enter
Type "EOF"
Enter
Sleep 500ms

Type "hystak sync myproject"
Enter
Sleep 1s

Type "cat $PROJECT_DIR/.mcp.json"
Enter
Sleep 1s

Type "hystak diff myproject"
Enter
Sleep 1s

Type "rm -rf $HYSTAK_CONFIG_DIR $PROJECT_DIR"
Enter
Sleep 500ms
