# Test: hystak --help displays all commands
# Verifies the CLI help output contains expected commands.

Output e2e/actual/help.gif

Set Shell "bash"
Set Width 100
Set Height 30

Type "hystak --help"
Enter
Sleep 1s

# Verify key commands are listed
Type "hystak list --help"
Enter
Sleep 1s

Type "hystak sync --help"
Enter
Sleep 1s
