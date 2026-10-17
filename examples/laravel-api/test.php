<?php

require_once __DIR__ . '/../../adapters/php-laravel/src/GoLoadTest.php';

use GoLoadTest\GoLoadTest;

echo "⚡ Running GoLoadTest via PHP/Laravel adapter...\n";

try {
    $result = GoLoadTest::run([
        'name' => 'Laravel User Endpoints Benchmark',
        'url' => 'http://localhost:8000/api/users',
        'duration' => '5s',
        'concurrency' => 20,
        'thresholds' => [
            'maxP95' => '100ms',
            'maxErrorRate' => 1.0,
        ],
    ]);

    echo "\n🎉 PHP Load Test Completed Successfully!\n";
    echo "- Total Requests: " . $result['totalRequests'] . "\n";
    echo "- Throughput: " . round($result['throughputRps'], 1) . " RPS\n";
    echo "- Passed: " . ($result['passed'] ? 'YES' : 'NO') . "\n";
} catch (\Exception $e) {
    echo "Load test notice: " . $e->getMessage() . "\n";
}
