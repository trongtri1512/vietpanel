<?php
/*
    OpenAdmin API client used by the OpenPanel WISECP module.

    OpenAdmin (the admin API of OpenPanel, https://openpanel.com) authenticates
    with username/password against POST /api/ and returns a short-lived JWT
    ("access_token") that must be sent as "Authorization: Bearer <token>" on
    every subsequent request. There is no static API key, so this class logs
    in lazily on first use and reuses the token for the lifetime of the
    request.

    API reference: https://dev.openpanel.com/api/
*/
class OpenPanelAPI
{
    private $server;
    private $token;

    public $error;
    public $status_code = 0;
    public $response;

    public function __construct($server = [])
    {
        $this->server = $server;
    }

    public function getHostname()
    {
        $protocol = !empty($this->server['secure']) ? 'https://' : 'http://';
        $port     = !empty($this->server['port']) ? $this->server['port'] : 2087;
        return $protocol . $this->server['ip'] . ':' . $port;
    }

    private function curl($url, $method, array $headers, $data = null)
    {
        $ch = curl_init();
        curl_setopt_array($ch, [
            CURLOPT_URL            => $url,
            CURLOPT_CUSTOMREQUEST  => $method,
            CURLOPT_HTTPHEADER     => $headers,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_SSL_VERIFYHOST => 0,
            CURLOPT_SSL_VERIFYPEER => 0,
            CURLOPT_TIMEOUT        => 30,
        ]);

        if ($data !== null) {
            curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
        }

        $body       = curl_exec($ch);
        $httpCode   = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $curlError  = curl_error($ch);
        curl_close($ch);

        $this->status_code = $httpCode;
        $this->response     = $body;

        if (class_exists('Modules')) {
            Modules::save_log('Servers', 'OpenPanel', $method . ' ' . $url, $data, $body ?: $curlError);
        }

        if ($body === false) {
            $this->error = 'cURL error: ' . $curlError;
            return false;
        }

        $decoded = json_decode($body, true);

        if ($httpCode < 200 || $httpCode >= 300) {
            $this->error = ($decoded['error'] ?? $decoded['message'] ?? null) ?: ('HTTP ' . $httpCode);
            return $decoded !== null ? $decoded : false;
        }

        return $decoded;
    }

    /*
        Authenticates with the OpenAdmin server credentials and caches the
        resulting JWT for the remainder of the request. Returns false (and
        sets $this->error) on failure.
    */
    private function authenticate()
    {
        if ($this->token) {
            return $this->token;
        }

        $response = $this->curl(
            $this->getHostname() . '/api/',
            'POST',
            ['Content-Type: application/json'],
            [
                'username' => $this->server['username'] ?? '',
                'password' => $this->server['password'] ?? '',
            ]
        );

        if (empty($response['access_token'])) {
            $this->error = 'Authentication with OpenAdmin failed: ' . ($this->error ?: 'no access_token returned');
            return false;
        }

        $this->token = $response['access_token'];
        return $this->token;
    }

    /*
        Performs an authenticated request against the OpenAdmin API.
        $uri must start with "/api/...". Returns the decoded JSON body
        (array) on success, or false on failure (see $this->error).
    */
    public function call($uri, $method = 'GET', $data = null)
    {
        $this->error = null;

        if (!$this->authenticate()) {
            return false;
        }

        $response = $this->curl(
            $this->getHostname() . $uri,
            $method,
            [
                'Authorization: Bearer ' . $this->token,
                'Content-Type: application/json',
            ],
            $data
        );

        if ($response === false) {
            return false;
        }

        return $response;
    }
}
