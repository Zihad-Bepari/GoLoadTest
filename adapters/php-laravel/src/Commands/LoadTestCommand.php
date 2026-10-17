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
                            {--url=http://localhost:8000/api/health : Target API URL}
                            {--duration=10s : Duration of the load test (e.g. 10s, 1m)}
                            {--concurrency=10 : Number of concurrent virtual users}
                            {--rps=0 : Requests per second rate limit}
                            {--max-p95=300ms : SLA Max P95 Latency threshold}
                            {--max-error=2.0 : SLA Max Error Rate percentage}';

    /**
     * The console command description.
     */
    protected $description = 'Run high-performance API load test using GoLoadTest core engine';

    /**
     * Execute the console command.
     */
    public function handle(): int
    {
        $this->info("Starting GoLoadTest against: " . $this->option('url'));

        try {
            $result = GoLoadTest::run([
                'url' => $this->option('url'),
                'duration' => $this->option('duration'),
                'concurrency' => (int) $this->option('concurrency'),
                'targetRps' => (float) $this->option('rps'),
                'maxP95' => $this->option('max-p95'),
                'maxErrorRate' => (float) $this->option('max-error'),
            ]);

            $passed = $result['passed'] ?? false;
            if ($passed) {
                $this->info("✓ Load test passed successfully! Total requests: {$result['totalRequests']}, RPS: {$result['throughputRps']}");
                return 0;
            } else {
                $this->error("✗ Load test failed! SLA thresholds breached.");
                return 1;
            }
        } catch (\Exception $e) {
            $this->error("Error running load test: " . $e->getMessage());
            return 1;
        }
    }
}
