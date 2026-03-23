Output e2e/actual/doctor.gif

Set Shell "bash"
Set Width 120
Set Height 120

Type "export HYSTAK_CONFIG_DIR=$(mktemp -d)"
Enter
Sleep 500ms

Type "mkdir -p $HYSTAK_CONFIG_DIR/profiles"
Enter
Sleep 500ms

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

Type "cat > $HYSTAK_CONFIG_DIR/projects.yaml << 'EOF'"
Enter
Type "projects: {}"
Enter
Type "EOF"
Enter
Sleep 500ms

Type "hystak doctor"
Enter
Sleep 1s

Type "rm -rf $HYSTAK_CONFIG_DIR"
Enter
Sleep 500ms
