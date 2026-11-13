<?php

namespace GoLoadTest;

use RuntimeException;

class GoLoadTest
{
    /**
     * Resolve the path to the GoLoadTest binary.
     */
    public static function resolveBinary(): string
    {
        $isWindows = strtoupper(substr(PHP_OS, 0, 3)) === 'WIN';
        $binaryName = $isWindows ? 'goloadtest.exe' : 'goloadtest';
        $arch = in_array(php_uname('m'), ['aarch64', 'arm64']) ? 'arm64' : 'amd64';
        $targetBinary = $isWindows ? 'goloadtest-windows-amd64.exe' : (PHP_OS_FAMILY === 'Darwin' ? "goloadtest-darwin-{$arch}" : "goloadtest-linux-{$arch}");

        $candidates = [
            getcwd() . DIRECTORY_SEPARATOR . $binaryName,
            getcwd() . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $targetBinary,
            getcwd() . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 2) . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 2) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $targetBinary,
            dirname(__DIR__, 2) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 3) . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 3) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $targetBinary,
            dirname(__DIR__, 3) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 4) . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 4) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $targetBinary,
            dirname(__DIR__, 4) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 5) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 6) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $targetBinary,
            dirname(__DIR__, 6) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $binaryName,
        ];

        foreach ($candidates as $candidate) {
            if (file_exists($candidate)) {
                if (PHP_OS_FAMILY !== 'Windows' && !is_executable($candidate)) {
                    @chmod($candidate, 0755);
                }
                // Auto-clear macOS Gatekeeper quarantine if on macOS
                if (PHP_OS_FAMILY === 'Darwin') {
                    @exec('xattr -d com.apple.quarantine ' . escapeshellarg($candidate) . ' 2>/dev/null');
                }
                return $candidate;
            }
        }

        return $binaryName; // fallback to PATH
    }

    // Run load test against API endpoints
    public static function run(array $options): array
    {
        $binary = self::resolveBinary();
        $outputDir = $options['outputPath'] ?? (getcwd() . DIRECTORY_SEPARATOR . 'results');

        if (!is_dir($outputDir)) {
            mkdir($outputDir, 0755, true);
        }

        $tempConfig = tempnam(sys_get_temp_dir(), 'glt_') . '.json';

        $breakUntilCrash = !empty($options['breakUntilCrash']) || (($options['pattern'] ?? '') === 'break-to-crash');
        $pattern = $breakUntilCrash ? 'break-to-crash' : ($options['pattern'] ?? 'constant');

        $stressConfig = [
            'enabled' => $breakUntilCrash,
            'initialConcurrency' => $options['initialConcurrency'] ?? 5,
            'stepConcurrency' => $options['stepConcurrency'] ?? 15,
            'stepInterval' => $options['stepInterval'] ?? '2s',
            'crashErrorRate' => $options['crashThreshold'] ?? 50.0,
            'crashConsecutiveFails' => 25,
            'crashMaxLatency' => '10s',
        ];

        $config = [
            'name' => $options['name'] ?? 'PHP/Laravel API Load Test',
            'baseUrl' => $options['baseUrl'] ?? '',
            'endpoints' => $options['endpoints'] ?? [
                [
                    'name' => $options['name'] ?? 'Default Endpoint',
                    'url' => $options['url'] ?? 'http://localhost:8000',
                    'method' => $options['method'] ?? 'GET',
                    'headers' => $options['headers'] ?? [],
                    'body' => is_array($options['body'] ?? '') ? json_encode($options['body']) : ($options['body'] ?? ''),
                    'timeout' => $options['timeout'] ?? '10s',
                    'expectedStatus' => $options['expectedStatus'] ?? [200],
                ]
            ],
            'duration' => $options['duration'] ?? '15s',
            'concurrency' => $options['concurrency'] ?? ($breakUntilCrash ? 200 : 10),
            'targetRps' => $options['targetRps'] ?? 0,
            'pattern' => $pattern,
            'stress' => $stressConfig,
            'thresholds' => $options['thresholds'] ?? [
                'maxP95' => $options['maxP95'] ?? '300ms',
                'maxErrorRate' => $options['maxErrorRate'] ?? 2.0,
            ],
            'circuitBreaker' => [
                'enabled' => true,
                'maxErrorRate' => 50.0,
            ],
            'monitor' => ['enabled' => true, 'localHost' => true],
            'outputFormat' => 'json',
            'outputPath' => $outputDir,
        ];

        file_put_contents($tempConfig, json_encode($config, JSON_PRETTY_PRINT));

        if (!function_exists('exec')) {
            @unlink($tempConfig);
            return self::runInProcess($options);
        }

        $cmd = escapeshellcmd($binary) . ' run --config ' . escapeshellarg($tempConfig) . ' --output ' . escapeshellarg($outputDir) . ' --format json 2>&1';
        
        $output = [];
        $exitCode = 0;
        @exec($cmd, $output, $exitCode);

        @unlink($tempConfig);

        // Find the generated JSON file
        $files = glob($outputDir . DIRECTORY_SEPARATOR . 'loadtest-*.json');
        if (!empty($files)) {
            rsort($files);
            $latestFile = $files[0];
            $content = file_get_contents($latestFile);
            $result = json_decode($content, true);
            $result['exitCode'] = $exitCode;
            return $result;
        }

        // If binary failed (e.g. noexec mount on cPanel or architecture incompatibility), seamlessly fallback to in-process engine
        return self::runInProcess($options);
    }

    // In-process HTTP load test fallback (curl_multi)
    public static function runInProcess(array $options): array
    {
        $endpoints = $options['endpoints'] ?? [];
        if (empty($endpoints) && !empty($options['url'])) {
            $endpoints = [['name' => $options['name'] ?? 'Default', 'url' => $options['url'], 'method' => $options['method'] ?? 'GET']];
        }
        if (empty($endpoints)) {
            return ['totalRequests' => 0, 'throughputRps' => 0, 'abortReason' => 'No endpoints specified'];
        }

        $durationSec = 5;
        if (!empty($options['duration'])) {
            $d = strtolower(trim((string)$options['duration']));
            if (substr($d, -1) === 's') $durationSec = max(1, (int)$d);
            elseif (substr($d, -1) === 'm') $durationSec = max(1, (int)$d * 60);
        }
        $concurrency = min(50, (int)($options['concurrency'] ?? 10));
        $startTime = microtime(true);
        $endTime = $startTime + $durationSec;

        $totalRequests = 0;
        $successful = 0;
        $failed = 0;
        $latencies = [];
        $statusCodes = [];

        if (function_exists('curl_multi_init')) {
            $mh = curl_multi_init();
            $handles = [];

            $addRequest = function() use (&$mh, &$handles, $endpoints) {
                $ep = $endpoints[array_rand($endpoints)];
                $ch = curl_init();
                curl_setopt($ch, CURLOPT_URL, $ep['url']);
                curl_setopt($ch, CURLOPT_CUSTOMREQUEST, $ep['method'] ?? 'GET');
                curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
                curl_setopt($ch, CURLOPT_TIMEOUT, 5);
                curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
                curl_setopt($ch, CURLOPT_SSL_VERIFYHOST, false);
                curl_setopt($ch, CURLOPT_USERAGENT, 'GoLoadTest-PHPEngine/1.0');
                curl_multi_add_handle($mh, $ch);
                $handles[(int)$ch] = ['ch' => $ch, 'start' => microtime(true)];
            };

            for ($i = 0; $i < $concurrency; $i++) {
                $addRequest();
            }

            do {
                curl_multi_exec($mh, $running);
                while ($info = curl_multi_info_read($mh)) {
                    $ch = $info['handle'];
                    $id = (int)$ch;
                    $start = $handles[$id]['start'] ?? microtime(true);
                    $latNs = (int)((microtime(true) - $start) * 1000000000);
                    $latencies[] = $latNs;
                    $totalRequests++;

                    $code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
                    $statusCodes[$code] = ($statusCodes[$code] ?? 0) + 1;
                    if ($code >= 200 && $code < 400) {
                        $successful++;
                    } else {
                        $failed++;
                    }

                    curl_multi_remove_handle($mh, $ch);
                    curl_close($ch);
                    unset($handles[$id]);

                    if (microtime(true) < $endTime) {
                        $addRequest();
                    }
                }
                if ($running) {
                    curl_multi_select($mh, 0.05);
                }
            } while ($running > 0 && microtime(true) < $endTime);

            foreach ($handles as $h) {
                curl_multi_remove_handle($mh, $h['ch']);
                curl_close($h['ch']);
            }
            curl_multi_close($mh);
        }

        $elapsed = max(0.1, microtime(true) - $startTime);
        sort($latencies);
        $count = count($latencies);
        $p95 = $count > 0 ? $latencies[(int)($count * 0.95)] : 0;
        $avg = $count > 0 ? (int)(array_sum($latencies) / $count) : 0;

        return [
            'suiteName' => $options['name'] ?? 'PHP In-Process Load Test',
            'duration' => (int)($elapsed * 1000000000),
            'totalRequests' => $totalRequests,
            'successRequests' => $successful,
            'failedRequests' => $failed,
            'errorRate' => $totalRequests > 0 ? round(($failed / $totalRequests) * 100, 2) : 0,
            'throughputRps' => round($totalRequests / $elapsed, 1),
            'latency' => [
                'min' => $latencies[0] ?? 0,
                'max' => end($latencies) ?: 0,
                'mean' => $avg,
                'median' => $count > 0 ? $latencies[(int)($count * 0.5)] : 0,
                'p90' => $count > 0 ? $latencies[(int)($count * 0.9)] : 0,
                'p95' => $p95,
                'p99' => $count > 0 ? $latencies[(int)($count * 0.99)] : 0,
            ],
            'statusCodes' => $statusCodes,
            'passed' => $failed === 0,
            'engine' => 'php-in-process-fallback'
        ];
    }
}
