# VietPanel WISECP Module 😎
WISECP hosting panel module for [VietPanel](https://github.com/trongtri1512/vietpanel)

---

## Usage

### Requirements

- Server with VietPanel Enterprise license
- WISECP

### Installation

1. Login to SSH for the WISECP server
2. Navigate to the WISECP install directory (e.g. `/var/www/wisecp`)
3. Clone this repository and copy its `coremio` folder into place:
   ```bash
   git clone https://github.com/stefanpejcic/openpanel-wisecp-module.git /tmp/openpanel-wisecp-module
   cp -r /tmp/openpanel-wisecp-module/coremio/modules/Servers/VietPanel coremio/modules/Servers/
   ```
4. In the WISECP admin panel, go to **Settings > Servers > Add New** and select **VietPanel** as the module

### Supported Operations

From WISECP admin area:
- Create a new account + add domain
- Suspend account
- Unsuspend account
- Change plan (upgrade/downgrade)
- Change password
- Terminate account
- View account information (plan, disk usage, domains) on the order page
- Login to VietAdmin (root access)

From WISECP client area:
- Auto-login as user
- Disk usage / bandwidth limit display

### Configuration

1. Add a new server in WISECP with the server's hostname/IP, the VietAdmin port (default `2087`), and an VietAdmin username/password that can authenticate against the API
2. Assign the server to a product, then open the product's module options and pick which **VietPanel Plan** it should provision accounts into (the dropdown is populated live from `/api/plans` on the assigned server)

### Troubleshooting

On WISECP:
1. Failed API calls are recorded through `Modules::save_log()` — check **Settings > Others > Module Log** (or the `mod_logs` table) for the request URL, payload and response
2. Errors returned by the module (`$this->error`) are shown directly on the relevant admin action

On VietPanel server:
1. [Enable `DEV_MODE` on VietAdmin](https://github.com/trongtri1512/vietpanel)
2. Trigger the action from WISECP
3. View the logs in: `/var/log/openpanel/admin/api.log`

### Update

1. Login to SSH for the WISECP server
2. Pull the latest files from this repository and copy `coremio/modules/Servers/VietPanel` over the existing installed copy

## Bug Reports

Report [new issue on github](https://github.com/stefanpejcic/openpanel-wisecp-module/issues/new/choose)
