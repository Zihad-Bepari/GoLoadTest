<?php

namespace GoLoadTest\Commands;

use Illuminate\Console\Command;
use GoLoadTest\GoLoadTest;

class LoadTestCommand extends Command
{
    /**
     * The name and signature of the console command.
     */
    protected $signature = 'test:load 
                            {--url= : Target API URL (leave empty for automatic discovery of all routes)}
                            {--duration=20s : Duration of the load test (e.g. 10s, 30s)}
                            {--concurrency=50 : Number of concurrent virtual users}
                            {--break-to-crash : Step up traffic until server breakdown (default in auto mode)}
                            {--rps=0 : Requests per second rate limit}
                            {--max-p95=300ms : SLA Max P95 Latency threshold}
                            {--max-error=2.0 : SLA Max Error Rate percentage}';

    /**
     * The console command description.
     */
    protected $description = 'Run autonomous API load & break-to-crash test using GoLoadTest core engine';

    /**
     * Execute the console command.
     */
    public function handle(): int
    {
        $targetUrl = $this->option('url');
        $baseUrl = rtrim(config('app.url') ?: 'http://127.0.0.1:8000', '/');
        $breakUntilCrash = $this->option('break-to-crash') || empty($targetUrl);

        $endpoints = [];

        if (empty($targetUrl)) {
            $this->info("⚡ GoLoadTest: Auto-discovering registered Laravel routes...");
            $router = app('router');
            $routes = $router->getRoutes();

            $discovered = [];
            foreach ($routes as $route) {
                $uri = $route->uri();
                // Filter out internal and debug endpoints
                if (str_starts_with($uri, '_') || str_contains($uri, 'sanctum') || str_contains($uri, 'telescope') || str_contains($uri, 'horizon')) {
                    continue;
                }

                $methods = array_diff($route->methods(), ['HEAD']);
                foreach ($methods as $method) {
                    if (in_array($method, ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'])) {
                        // Normalize parameterized routes: /users/{id} -> /users/1
                        $cleanUri = preg_replace('/\{[a-zA-Z0-9_]+\??\}/', '1', $uri);
                        $fullUrl = $baseUrl . '/' . ltrim($cleanUri, '/');
                        $endpoints[] = [
                            'name' => "{$method} /{$uri}",
                            'url' => $fullUrl,
                            'method' => $method,
                            'timeout' => '8s',
                        ];
                        $discovered[] = [$method, "/{$uri}", $fullUrl];
                    }
                }
            }

            if (empty($endpoints)) {
                $this->warn("No API routes auto-detected. Defaulting to {$baseUrl}/api/health");
                $endpoints[] = [
                    'name' => 'Health Check',
                    'url' => "{$baseUrl}/api/health",
                    'method' => 'GET',
                ];
            } else {
                $this->table(['Method', 'Route URI', 'Resolved Target URL'], array_slice($discovered, 0, 10));
                if (count($discovered) > 10) {
                    $this->info("... and " . (count($discovered) - 10) . " more routes (Total: " . count($discovered) . " routes discovered)");
                }
            }
        } else {
            $this->info("Starting GoLoadTest against specific target: " . $targetUrl);
            $endpoints[] = [
                'name' => 'Target API',
                'url' => $targetUrl,
                'method' => 'GET',
            ];
        }

        try {
            $this->info("🚀 Initiating load test (Break-Until-Crash: " . ($breakUntilCrash ? "YES" : "NO") . ")...");
            $result = GoLoadTest::run([
                'name' => 'Laravel Autonomous API Stress Test',
                'baseUrl' => $baseUrl,
                'endpoints' => $endpoints,
                'duration' => $this->option('duration'),
                'concurrency' => (int) $this->option('concurrency'),
                'breakUntilCrash' => $breakUntilCrash,
                'targetRps' => (float) $this->option('rps'),
                'maxP95' => $this->option('max-p95'),
                'maxErrorRate' => (float) $this->option('max-error'),
            ]);

            $this->newLine();
            $this->info("=== GOLOADTEST PERFORMANCE REPORT ===");
            $this->line("- Total Requests: " . ($result['totalRequests'] ?? 0));
            $this->line("- Achieved Throughput: " . round($result['throughputRps'] ?? 0, 1) . " RPS");
            $this->line("- Error Rate: " . round($result['errorRate'] ?? 0, 2) . "%");

            if (!empty($result['breakingPoint']['detected'])) {
                $this->warn("⚠️  BREAKING POINT REACHED:");
                $this->line("   - Crash Concurrency: " . $result['breakingPoint']['crashConcurrency'] . " users");
                $this->line("   - Safe Stable Concurrency: " . $result['breakingPoint']['maxStableConcurrency'] . " users");
                $this->line("   - Reason: " . $result['breakingPoint']['crashReason']);
            }

            if (!empty($result['endpoints'])) {
                $table = [];
                foreach ($result['endpoints'] as $ep) {
                    $p95Ms = round(($ep['latency']['p95'] ?? 0) / 1000000, 1) . 'ms';
                    $table[] = [
                        $ep['name'],
                        $ep['totalRequests'],
                        round($ep['rps'], 1) . ' req/s',
                        $p95Ms,
                        round($ep['errorRate'], 1) . '%'
                    ];
                }
                $this->table(['Endpoint', 'Requests', 'RPS', 'P95 Latency', 'Error %'], $table);
            }

            $passed = $result['passed'] ?? false;
            if ($passed) {
                $this->info("✓ Load test passed successfully!");
                return 0;
            } else {
                $this->error("✗ Load test finished with SLA breaches or crash.");
                return 1;
            }
        } catch (\Exception $e) {
            $this->error("Error running load test: " . $e->getMessage());
            return 1;
        }
    }
}
