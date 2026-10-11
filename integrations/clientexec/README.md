# VietPanel ClientExec Plugin 😎
ClientExec server plugin for [VietPanel](https://github.com/trongtri1512/vietpanel)

---

## Usage

### Requirements

- Server with VietPanel Enterprise license
- ClientExec

### Installation

1. Login to SSH for the ClientExec server
2. Navigate to `path_to_clientexec/plugins/server`
3. Run this command to create a new folder and in it download the plugin:
   ```bash
   git clone https://github.com/stefanpejcic/openpanel-clientexec.git openpanel
   ```
4. In ClientExec admin, add a new server and select the **VietPanel** plugin
5. Fill in the VietAdmin admin username/password, hostname, and port (default `2087`)

### Supported Operations

From ClientExec admin area:
- Create a new user account + add domain
- Suspend account
- Unsuspend account
- Change package
- Change password
- Terminate account
- Test connection

### Configuration

The plugin authenticates to the VietAdmin API (`/api/`) with the admin username/password
configured on the server, and uses the resulting JWT bearer token for all subsequent
calls to `/api/users/<username>` and `/api/domains/new`.

### Troubleshooting

On ClientExec:
1. Enable debug logging for the server module and reproduce the action
2. Check the module log in the ClientExec admin area

On VietPanel server:
1. [Enable `DEV_MODE` on VietAdmin](https://github.com/trongtri1512/vietpanel)
2. Send requests from the ClientExec server
3. View the logs in: `/var/log/openpanel/admin/api.log`

### Update

1. Login to SSH for the ClientExec server
2. Navigate to `path_to_clientexec/plugins/server/openpanel`
3. Run this command to download newer files:
   ```bash
   git pull
   ```

## Bug Reports

Report [new issue on github](https://github.com/stefanpejcic/openpanel-clientexec/issues/new/choose)
