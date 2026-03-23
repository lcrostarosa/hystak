Output e2e/actual/sync_workflow.gif

Set Shell "bash"
Set Width 120
Set Height 120

# Use a fixed path to avoid $ expansion issues in VHS Type commands
Type "export HYSTAK_CONFIG_DIR=/tmp/hystak-e2e-test"
Enter
Type "export PROJECT_DIR=/tmp/hystak-e2e-project"
Enter
Sleep 500ms

Type "mkdir -p /tmp/hystak-e2e-test/profiles /tmp/hystak-e2e-project/.git"
Enter
Sleep 500ms

# Create minimal registry
Type "cat > /tmp/hystak-e2e-test/registry.yaml << 'EOF'"
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
Type "cat > /tmp/hystak-e2e-test/projects.yaml << 'EOF'"
Enter
Type "projects:"
Enter
Type "  myproject:"
Enter
Type "    path: /tmp/hystak-e2e-project"
Enter
Type "    active_profile: dev"
Enter
Type "    managed_mcps: []"
Enter
Type "EOF"
Enter
Sleep 500ms

# Create profile
Type "cat > /tmp/hystak-e2e-test/profiles/dev.yaml << 'EOF'"
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

Type "cat /tmp/hystak-e2e-project/.mcp.json"
Enter
Sleep 1s

Type "hystak diff myproject"
Enter
Sleep 1s

Type "rm -rf /tmp/hystak-e2e-test /tmp/hystak-e2e-project"
Enter
Sleep 500ms
