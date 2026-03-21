# E2E test: verify hystak list output with seeded registry
Output e2e/actual/list_servers.txt
Set Shell "bash"
Set Width 100
Set Height 30

Type "hystak --config-dir $HYSTAK_CONFIG_DIR list"
Enter
Sleep 1s
