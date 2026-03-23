# Test: hystak completion generates shell completions
# Verifies bash, zsh, and fish completions can be generated.

Output e2e/actual/completion.gif

Set Shell "bash"
Set Width 80
Set Height 15

Type "hystak completion bash | head -5"
Enter
Sleep 1s

Type "hystak completion zsh | head -5"
Enter
Sleep 1s

Type "hystak completion fish | head -5"
Enter
Sleep 1s
