<?php
    return [
        'type'                          => "hosting",
        'access-hash'                   => false,
        'server-info-checker'           => false,
        'server-info-port'              => true,
        'server-info-not-secure-port'   => 2087,
        'server-info-secure-port'       => 2087,
        'supported' => [
            'disk-bandwidth-usage',
            'change-password',
        ],
        'configurable-option-params'    => [
            'plan',
        ],
    ];