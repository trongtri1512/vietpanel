<?php
/*
Plugin Name: Force SSL behind Varnish
Description: Forces SSL for wp-admin, to avoid CORS errors behind Varnish Caching in VietPanel.
Version: 1.0
Plugin URI: https://github.com/trongtri1512/vietpanel/articles/docker/https-wordpress-behind-varnish-proxy/
Author: Stefan Pejcic
*/

defined('ABSPATH') || exit;
define('FORCE_SSL_ADMIN', true);

add_action('init', function() {
    if (isset($_SERVER['HTTP_X_FORWARDED_PROTO']) && $_SERVER['HTTP_X_FORWARDED_PROTO'] === 'https') {
        $_SERVER['HTTPS'] = 'on';
    }
});
