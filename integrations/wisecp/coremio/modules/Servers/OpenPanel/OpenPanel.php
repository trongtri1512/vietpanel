<?php
    /*
        OpenPanel WISECP Module
        Usage: https://openpanel.com/docs/articles/extensions/openpanel-and-wisecp/
        Source: https://github.com/stefanpejcic/openpanel-wisecp-module
        Talks to OpenAdmin (OpenPanel's admin API) over HTTPS/HTTP using the
        server's username + password to obtain a JWT, then drives account
        lifecycle actions through /api/users/{username}. See api.class.php.
    */
    class OpenPanel_Module extends ServerModule
    {
        public $api;

        function __construct($server,$options=[])
        {
            $this->_name = __CLASS__;

            // If you are developing, set the $this->force_setup value to be false.
            $this->force_setup  = false; // If "true", even if the module gets an error message, it will be ignored and the installation is complete.
            parent::__construct($server,$options);
        }

        protected function define_server_info($server=[])
        {
            if(!class_exists("OpenPanelAPI")) include __DIR__.DS."api.class.php";
            $this->api    = new OpenPanelAPI($server);
            $this->server = $server;
        }

        public function testConnect(){
            if (!$this->api->call('/api/plans', 'GET')) {
                $this->error = $this->api->error ?: 'Unable to connect to OpenAdmin API';
                return false;
            }

            return true;
        }

        /*
            Populated with the plans configured on the OpenPanel server, so the
            admin can pick which OpenPanel plan a WISECP product provisions
            into. Selected value is stored under creation_info['plan'] (see
            config.php -> configurable-option-params) and read back in
            create()/apply_updowngrade().
        */
        public function config_options($data=[])
        {
            $options = [];

            $response = $this->api ? $this->api->call('/api/plans', 'GET') : false;
            if ($response && !empty($response['plans'])) {
                foreach ($response['plans'] as $plan) {
                    $options[$plan['name']] = $plan['name'];
                }
            }

            if (!$options) {
                return [
                    'Note' => [
                        'wrap_width'  => 100,
                        'name'        => 'Note',
                        'type'        => 'output',
                        'value'       => 'Could not load plans from OpenPanel. Save the server connection details first, then reopen this page.',
                    ],
                ];
            }

            return [
                'plan' => [
                    'wrap_width'  => 100,
                    'name'        => 'OpenPanel Plan',
                    'description' => 'The OpenPanel plan to assign to accounts created for this product.',
                    'type'        => 'dropdown',
                    'options'     => $options,
                    'value'       => isset($data['plan']) ? $data['plan'] : key($options),
                ],
            ];
        }

        public function generate_username($domain='',$half_mixed=false){
            $exp            = explode(".",$domain);
            $domain         = Filter::transliterate($exp[0]);
            $username       = $domain;
            $fchar          = substr($username,0,1);
            $size           = strlen($username);
            if($fchar == "0" || (int)$fchar)
                $username   = Utility::generate_hash(1,false,"l").substr($username,1,$size-1);

            if($size>=8){
                $username   = substr($username,0,5);
                $username .= Utility::generate_hash(3,false,"l");
            }elseif($size>4 && $size<9){
                $username   = substr($username,0,5);
                $username .= Utility::generate_hash(3,false,"l");
            }elseif($size>=1 && $size<5){
                $how        = (8 - $size);
                $username   = substr($username,0,$size);
                $username .= Utility::generate_hash($how,false,"l");
            }

            return $username;
        }

        public function create($domain = '',array $order_options=[])
        {
            $username       = $this->generate_username($domain);
            $password       = Utility::generate_hash(12);

            if(isset($order_options["username"]) && $order_options["username"]) $username = $order_options["username"];
            if(isset($order_options["password"]) && $order_options["password"]) $password = $order_options["password"];
            $username       = str_replace("-","",$username);

            $creationInfo   = isset($order_options["creation_info"]) ? $order_options["creation_info"] : [];
            $planName       = isset($creationInfo["plan"]) ? $creationInfo["plan"] : null;
            $email          = isset($this->user['email']) ? $this->user['email'] : '';

            if (!$planName) {
                $this->error = 'No OpenPanel plan selected for this product. Configure it under the product\'s module options.';
                return false;
            }

            try
            {
                $response = $this->api->call('/api/users', 'POST', [
                    'username'  => $username,
                    'password'  => $password,
                    'email'     => $email,
                    'plan_name' => $planName,
                ]);

                if (!$response || empty($response['success'])) {
                    throw new Exception($this->api->error ?: ($response['error'] ?? 'Failed to create OpenPanel account'));
                }

                if ($domain) {
                    $domainResponse = $this->api->call('/api/domains/new', 'POST', [
                        'username' => $username,
                        'domain'   => $domain,
                        'docroot'  => '/var/www/html/' . $domain,
                    ]);

                    if (!$domainResponse || ($domainResponse['returncode'] ?? 1) !== 0) {
                        throw new Exception('Account created, but failed to add domain: ' . trim($domainResponse['stderr'] ?? ($this->api->error ?: 'Unknown error')));
                    }
                }
            }
            catch (Exception $e){
                $this->error = $e->getMessage();
                self::save_log(
                    'Servers',
                    $this->_name,
                    __FUNCTION__,
                    ['order' => $this->order],
                    $e->getMessage(),
                    $e->getTraceAsString()
                );
                return false;
            }

            return [
                'username' => $username,
                'password' => $password,
            ];
        }

        public function suspend()
        {
            return $this->userAction('suspend');
        }

        public function unsuspend()
        {
            return $this->userAction('unsuspend');
        }

        public function terminate()
        {
            // OpenAdmin refuses to delete a suspended account's data, so
            // unsuspend first (ignore its result) and then delete.
            $this->api->call('/api/users/' . rawurlencode($this->config['user']), 'PATCH', ['action' => 'unsuspend']);

            $response = $this->api->call('/api/users/' . rawurlencode($this->config['user']), 'DELETE');

            if (!$response || empty($response['success'])) {
                $this->error = $this->api->error ?: ($response['error'] ?? 'Failed to terminate account');
                self::save_log('Servers', $this->_name, __FUNCTION__, ['order' => $this->order], $this->error, '');
                return false;
            }

            return true;
        }

        public function change_password($password=''){
            $response = $this->api->call('/api/users/' . rawurlencode($this->config['user']), 'PATCH', [
                'password' => $password,
            ]);

            if (!$response || empty($response['success'])) {
                $this->error = $this->api->error ?: ($response['error'] ?? 'Failed to change password');
                self::save_log('Servers', $this->_name, __FUNCTION__, ['order' => $this->order], $this->error, '');
                return false;
            }

            return true;
        }

        public function apply_updowngrade($orderopt=[],$product=[]){
            $creationInfo = isset($orderopt["creation_info"]) ? $orderopt["creation_info"] : [];
            $planName     = isset($creationInfo["plan"]) ? $creationInfo["plan"] : null;

            if (!$planName) {
                $this->error = 'No OpenPanel plan selected to switch this account to.';
                return false;
            }

            $response = $this->api->call('/api/users/' . rawurlencode($this->config['user']), 'PUT', [
                'plan_name' => $planName,
            ]);

            if (!$response || empty($response['success'])) {
                $this->error = $this->api->error ?: ($response['error'] ?? 'Failed to change plan');
                self::save_log('Servers', $this->_name, __FUNCTION__, ['order' => $this->order], $this->error, '');
                return false;
            }

            return true;
        }

        /*
            Runs a suspend/unsuspend PATCH against the account identified by
            $this->config['user'] (the "username" value returned by create()).
        */
        private function userAction($action)
        {
            $response = $this->api->call('/api/users/' . rawurlencode($this->config['user']), 'PATCH', [
                'action' => $action,
            ]);

            if (!$response || empty($response['success'])) {
                $this->error = $this->api->error ?: ($response['error'] ?? ('Failed to ' . $action . ' account'));
                self::save_log('Servers', $this->_name, __FUNCTION__, ['order' => $this->order], $this->error, '');
                return false;
            }

            return true;
        }

        private function fetchAccountInfo()
        {
            $response = $this->api->call('/api/users/' . rawurlencode($this->config['user']), 'GET');
            if (!$response || empty($response['user'])) {
                return null;
            }
            return $response['user'];
        }

        public function getDisk(){
            $account = $this->fetchAccountInfo();
            if (!$account) {
                $this->error = $this->api->error ?: 'Unable to fetch disk usage from OpenPanel';
                return false;
            }

            $disk  = $account['disk_usage'] ?? [];
            $plan  = $account['plan'] ?? [];
            $used  = (int)($disk['disk_used'] ?? 0);
            $limit = (int)($plan['disk_limit'] ?? ($disk['disk_hard'] ?? 0));

            $percent = ($limit > 0) ? Utility::getPercent($used,$limit) : 0;
            if($percent>100) $percent = 100;

            return [
                'limit' => $limit,
                'used'  => $used,
                'used-percent' => $percent,
                'format-limit' => $limit ? $limit.' MB' : "∞",
                'format-used' => $used." MB",
            ];
        }

        public function getBandwidth(){
            // OpenAdmin's API does not currently expose per-account bandwidth
            // usage, only the plan's bandwidth limit, so "used" always reports 0.
            $account = $this->fetchAccountInfo();
            if (!$account) {
                $this->error = $this->api->error ?: 'Unable to fetch bandwidth info from OpenPanel';
                return false;
            }

            $plan  = $account['plan'] ?? [];
            $limit = (int)($plan['bandwidth'] ?? 0);

            return [
                'limit' => $limit,
                'used'  => 0,
                'used-percent' => 0,
                'format-limit' => $limit ? $limit.' GB' : "∞",
                'format-used' => "0 GB",
            ];
        }

        public function clientArea()
        {
            $content    = $this->clientArea_buttons_output();
            $_page      = $this->page;

            if(!$_page) $_page = 'home';

            $loginHtml = '<p>Error generating autologin link.</p>';
            $response  = $this->api->call('/api/users/' . rawurlencode($this->config['user']) . '/autologin', 'POST');
            if ($response && !empty($response['link'])) {
                $loginHtml = '<a class="yesilbtn" href="' . htmlspecialchars($response['link']) . '" target="_blank">Login to OpenPanel</a>';
            }

            $content .= $this->get_page('clientArea-'.$_page,['login_html' => $loginHtml]);
            return  $content;
        }

        public function clientArea_buttons()
        {
            return [];
        }

        public function adminArea_buttons()
        {
            return [
                'openadmin_login' => [
                    'text' => 'Login to OpenAdmin',
                    'type' => 'function',
                    'target_blank' => true,
                ],
            ];
        }

        // Root/admin single sign-on via POST /api/login -> GET /login/sso/{token}
        public function use_adminArea_openadmin_login()
        {
            $response = $this->api->call('/api/login', 'POST', [
                'username' => $this->server['username'] ?? '',
                'password' => $this->server['password'] ?? '',
            ]);

            if (!$response || empty($response['login_path'])) {
                echo 'Unable to generate an OpenAdmin login link: ' . ($this->api->error ?: ($response['error'] ?? 'unknown error'));
                return false;
            }

            Utility::redirect($this->api->getHostname() . $response['login_path']);
            echo "Redirecting...";
        }

        /*
            SERVICES IN ADMIN AREA - read-only account snapshot pulled live
            from OpenAdmin, shown on the order/service detail page.
        */
        public function adminArea_service_fields(){
            $account = $this->fetchAccountInfo();

            if (!$account) {
                return [
                    'Note' => [
                        'wrap_width' => 100,
                        'name'       => 'OpenPanel Account',
                        'type'       => 'output',
                        'value'      => 'Could not load account information from OpenPanel: ' . ($this->api->error ?: 'user not found'),
                    ],
                ];
            }

            $plan = $account['plan'] ?? [];
            $disk = $account['disk_usage'] ?? [];

            return [
                'openpanel_plan' => [
                    'wrap_width' => 100,
                    'name'       => 'Plan',
                    'type'       => 'output',
                    'value'      => $plan['name'] ?? '-',
                ],
                'openpanel_disk' => [
                    'wrap_width' => 100,
                    'name'       => 'Disk Usage',
                    'type'       => 'output',
                    'value'      => ($disk['disk_used'] ?? 0) . ' / ' . ($plan['disk_limit'] ?? '?') . ' MB',
                ],
                'openpanel_domains' => [
                    'wrap_width' => 100,
                    'name'       => 'Domains',
                    'type'       => 'output',
                    'value'      => count($account['domains'] ?? []),
                ],
            ];
        }

    }
