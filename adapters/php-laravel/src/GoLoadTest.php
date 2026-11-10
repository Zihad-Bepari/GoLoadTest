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
            dirname(__DIR__, 2) . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 2) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $targetBinary,
            dirname(__DIR__, 2) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 4) . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 4) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $targetBinary,
            dirname(__DIR__, 4) . DIRECTORY_SEPARATOR . 'bin' . DIRECTORY_SEPARATOR . $binaryName,
        ];

        foreach ($candidates as $candidate) {
            if (file_exists($candidate)) {
                return $candidate;
            }
        }

        return $binaryName; // fallback to PATH
    }

    /**
     * Run load testing against an API endpoint.
     *
     * @param array $options Configuration options
     * @return array Decoded test results
     */
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

        $cmd = escapeshellcmd($binary) . ' run --config ' . escapeshellarg($tempConfig) . ' --output ' . escapeshellarg($outputDir) . ' --format json';
        
        $output = [];
        $exitCode = 0;
        exec($cmd, $output, $exitCode);

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

        if ($exitCode !== 0) {
            throw new RuntimeException("GoLoadTest failed with exit code {$exitCode}. SLA Thresholds breached or aborted.");
        }

        return ['status' => 'completed', 'exitCode' => $exitCode];
    }
}
