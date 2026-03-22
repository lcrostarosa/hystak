# E2E smoke test: TUI launches, shows tabs, and quits cleanly
Output e2e/actual/tui_startup.gif
Set Shell "bash"
Set Width 120
Set Height 35
Set FontSize 14

Type "hystak --config-dir $HYSTAK_CONFIG_DIR manage"
Enter
Sleep 1500ms

# Navigate through tabs (arrow keys)
Right
Sleep 500ms
Right
Sleep 500ms
Right
Sleep 500ms

# Quit
Type "q"
Sleep 500ms
