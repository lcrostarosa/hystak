Output e2e/actual/list_servers.gif

Set Shell "bash"
Set Width 120
Set Height 120

Type "export HYSTAK_CONFIG_DIR=$(mktemp -d)"
Enter
Sleep 500ms

Type "mkdir -p $HYSTAK_CONFIG_DIR"
Enter
Sleep 500ms

# Write registry using heredoc to avoid VHS special char issues
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
Type "EOF"
Enter
Sleep 500ms

Type "hystak list"
Enter
Sleep 1s

Type "rm -rf $HYSTAK_CONFIG_DIR"
Enter
Sleep 500ms
