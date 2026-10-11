# VietCLI  

VietCLI is the command-line interface for managing [VietPanel](https://github.com/trongtri1512/vietpanel/).  

On installation, the `vietcli.sh` script is added to the system path, and `vietcli --help` generates the list of all commands (`aliases.txt`).  

All scripts from `/usr/local/vietcli/` can be accessed using the `vietcli` command by replacing `/` with `-`.  

For example: to run user/add.sh script you would type: `vietcli user-add`.

## Tab completion

`lib/completion.bash` provides bash tab-completion for `vietcli` commands and their arguments (usernames, domains, plan names, PHP versions, and more, pulled live from the panel database and filesystem). It's symlinked into `/etc/bash_completion.d/vietcli` automatically whenever `vietcli update` runs, which also installs the `bash-completion` package if it isn't already present.

## Updates

VietPanel is proud of the modularity, so you can independently update just the VietCLI when needed.

To update VietCLI:

```sh
vietcli update --cli
```
