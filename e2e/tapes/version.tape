# Test: hystak version prints version info
# Verifies version, commit, and date fields are present.

Output e2e/actual/version.gif

Set Shell "bash"
Set Width 80
Set Height 10

Type "hystak version"
Enter
Sleep 1s
