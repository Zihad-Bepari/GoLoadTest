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

        $candidates = [
            getcwd() . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 2) . DIRECTORY_SEPARATOR . $binaryName,
            dirname(__DIR__, 4) . DIRECTORY_SEPARATOR . $binaryName,
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

        $config = [
            'name' => $options['name'] ?? 'PHP/Laravel API Load Test',
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
            'duration' => $options['duration'] ?? '10s',
            'concurrency' => $options['concurrency'] ?? 10,
            'targetRps' => $options['targetRps'] ?? 0,
            'pattern' => $options['pattern'] ?? 'constant',
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
