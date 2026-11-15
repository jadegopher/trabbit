# Run from the repository root against a disposable RabbitMQ broker.
# Prerequisites and the full recording workflow are in CONTRIBUTING.md.
Output docs/demo.gif

Require bash

Set Shell "bash"
Set FontSize 16
Set Width 1440
Set Height 900
Set Padding 20
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set BorderRadius 12
Set Framerate 12
Set TypingSpeed 35ms
Set CursorBlink false
Set LoopOffset 68%
Set WaitTimeout 15s

Hide
Type@0ms "unset NO_COLOR; export COLORTERM=truecolor PS1='$ '; clear"
Enter
Sleep 500ms
Show

Type "./trabbit --url 'http://localhost:15672' --user guest --pass guest"
Enter
Wait+Screen /Add q/
Sleep 1s

# Create the demo queue using the real management API.
Type "a"
Wait+Screen /Add New Queue/
Type "test"
Sleep 700ms
Enter
Sleep 2s
Type "f"
Wait+Screen /Filter Queues/
Type "test"
Enter
Sleep 2s

# Paste the supplied JSON, then format and edit it.
Type "p"
Wait+Screen /Payload/
Copy `{"overview":{},"connections":{},"channels":{},"exchanges":{},"queues":{"selected_subtab":"overview","table":{"page_size":100},"overview":{},"charts":{"style":"dotted","size":"medium","time_range":"10m","throughput":{"Ack":false,"Deliver":true,"DeliverGet":true,"DeliverNoAck":false,"DropUnroutable":false,"Get":false,"GetNoAck":false,"Publish":false,"Redeliver":false,"ReturnUnroutable":false}},"consumers":{"page_size":100},"bindings":{"page_size":100}},"admin":{}}`
Paste
Sleep 1s
Ctrl+B
Sleep 1s
Ctrl+[
Type@0ms "[1;5H"
Sleep 700ms
Down
# Shift+End selects the complete first field.
Ctrl+[
Type@0ms "[1;2F"
Sleep 1500ms
Delete
Sleep 500ms
Delete
Sleep 700ms
# Undo the newline deletion and field deletion, then redo both.
Ctrl+Z
Ctrl+Z
Sleep 1s
Ctrl+[
Type@0ms "[122;6u"
Ctrl+[
Type@0ms "[122;6u"
Sleep 1s
Ctrl+B
Sleep 1s
Ctrl+[
Type@0ms "[1;5H"
Ctrl+G
Wait+Screen /Text editing/
Sleep 2s
Escape
Sleep 500ms
# VHS has no F5 command; send its standard terminal escape sequence.
Ctrl+[
Type@0ms "[15~"
Wait+Screen /Published/
Sleep 2s
Escape
Wait+Screen /Discard draft/
Type "y"
Sleep 6s

# Filter live traffic, scroll the overview, and tour the detail subtabs.
Type "f"
Wait+Screen /Filter Queues/
Ctrl+E
Backspace 4
Type "stu"
Enter
Sleep 2s
Tab
Down@60ms 10
Sleep 2s
Down@60ms 10
Sleep 1s
Up@60ms 20
Sleep 1s
Type "]"
Sleep 2s
Type "]"
Sleep 2s
Type "]"
Sleep 2s
Type "]"
Sleep 2s

Ctrl+C
Sleep 700ms
Ctrl+C
Sleep 500ms
