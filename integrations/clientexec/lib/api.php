<?php

class OpenPanelAPI
{
    private $adminUser;
    private $adminPassword;
    private $hostname;
    private $port;
    private $token;

    public function __construct($adminUser, $adminPassword, $hostname, $port = 2087)
    {
        $this->adminUser = $adminUser;
        $this->adminPassword = $adminPassword;
        $this->hostname = $hostname;
        $this->port = $port ?: 2087;
    }

    private function baseUrl()
    {
        $protocol = filter_var($this->hostname, FILTER_VALIDATE_IP) ? 'http://' : 'https://';
        return $protocol . $this->hostname . ':' . $this->port;
    }

    private function applyCaBundle($ch)
    {
        $caPathOrFile = \Composer\CaBundle\CaBundle::getSystemCaRootBundlePath();
        if (is_dir($caPathOrFile)) {
            curl_setopt($ch, CURLOPT_CAPATH, $caPathOrFile);
        } else {
            curl_setopt($ch, CURLOPT_CAINFO, $caPathOrFile);
        }
    }

    private function authenticate()
    {
        if ($this->token) {
            return $this->token;
        }

        $ch = curl_init();
        $url = $this->baseUrl() . '/api/';

        CE_Lib::log(4, 'OpenAdmin Auth Request to: ' . $url);

        curl_setopt($ch, CURLOPT_URL, $url);
        curl_setopt($ch, CURLOPT_HTTPHEADER, array('Content-Type: application/json'));
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_POST, true);
        curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode([
            'username' => $this->adminUser,
            'password' => $this->adminPassword,
        ]));
        $this->applyCaBundle($ch);

        $result = curl_exec($ch);

        if (curl_errno($ch)) {
            $error = curl_error($ch);
            curl_close($ch);
            throw new CE_Exception('Connection error: ' . $error);
        }
        curl_close($ch);

        CE_Lib::log(4, 'OpenAdmin Auth Response: ' . $result);

        $data = json_decode($result, true);
        if (empty($data['access_token'])) {
            throw new CE_Exception('Authentication with OpenAdmin failed');
        }

        $this->token = $data['access_token'];
        return $this->token;
    }

    private function call($method, $uri, $data = null)
    {
        $token = $this->authenticate();

        $ch = curl_init();
        $url = $this->baseUrl() . $uri;

        CE_Lib::log(4, 'OpenPanel Request: ' . $method . ' ' . $url);
        if ($data !== null) {
            CE_Lib::log(4, 'Post Params: ');
            CE_Lib::log(4, $data);
        }

        curl_setopt($ch, CURLOPT_URL, $url);
        curl_setopt($ch, CURLOPT_CUSTOMREQUEST, $method);
        curl_setopt($ch, CURLOPT_HTTPHEADER, array(
            'Content-Type: application/json',
            'Authorization: Bearer ' . $token,
        ));
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        if ($data !== null) {
            curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
        }
        $this->applyCaBundle($ch);

        $result = curl_exec($ch);
        CE_Lib::log(4, 'OpenPanel Response: ');
        CE_Lib::log(4, $result);

        if (curl_errno($ch)) {
            $error = curl_error($ch);
            curl_close($ch);
            throw new CE_Exception('Connection error: ' . $error);
        }
        curl_close($ch);

        $result = json_decode($result, true);

        if (isset($result['success']) && $result['success'] === false) {
            throw new CE_Exception($result['message'] ?? $result['error'] ?? 'Unknown error from OpenPanel');
        }
        if (isset($result['error']) && $result['error'] != '') {
            throw new CE_Exception($result['error']);
        }

        return $result;
    }

    public function createAccount($username, $password, $email, $planName, $domainName = null)
    {
        $params = [
            'username' => $username,
            'password' => $password,
            'email' => $email,
            'plan_name' => $planName,
        ];
        $result = $this->call('POST', '/api/users/' . $username, $params);

        if (!empty($domainName)) {
            $this->addDomain($username, $domainName);
        }

        return $result;
    }

    public function addDomain($username, $domainName, $docroot = null)
    {
        $params = [
            'username' => $username,
            'domain' => $domainName,
            'docroot' => $docroot ?: '/var/www/html/' . $domainName,
        ];
        return $this->call('POST', '/api/domains/new', $params);
    }

    public function terminateAccount($username)
    {
        return $this->call('DELETE', '/api/users/' . $username);
    }

    public function suspendAccount($username)
    {
        return $this->call('PATCH', '/api/users/' . $username, ['action' => 'suspend']);
    }

    public function unsuspendAccount($username)
    {
        return $this->call('PATCH', '/api/users/' . $username, ['action' => 'unsuspend']);
    }

    public function changeAccountPassword($username, $password)
    {
        return $this->call('PATCH', '/api/users/' . $username, ['password' => $password]);
    }

    public function changeAccountPackage($username, $planName)
    {
        return $this->call('PUT', '/api/users/' . $username, ['plan_name' => $planName]);
    }

    public function getUser($username)
    {
        return $this->call('GET', '/api/users/' . $username);
    }

    public function verifyConnection()
    {
        return $this->authenticate();
    }
}
