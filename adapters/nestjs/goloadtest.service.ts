import { Injectable } from '@nestjs/common';
import { runLoadTest } from '../nodejs';
import type { LoadTestOptions, TestResult } from '../nodejs';

/**
 * GoLoadTestService — NestJS injectable service for high-performance load testing.
 *
 * Usage in a module:
 *   Import GoLoadTestModule into any NestJS module, then inject GoLoadTestService
 *   via constructor injection in your test class or service.
 *
 * Example (e2e test):
 *   const result = await this.loadTestService.run({
 *     url: 'http://localhost:3000/api/users',
 *     duration: '30s',
 *     concurrency: 50,
 *     thresholds: { maxP95: '200ms', maxErrorRate: 1.0 },
 *   });
 */
@Injectable()
export class GoLoadTestService {
  /**
   * Runs a load test against one or more API endpoints using the GoLoadTest core engine.
   * @param options - Test configuration: URL(s), duration, concurrency, SLA thresholds, etc.
   * @returns A promise that resolves with the full TestResult report.
   */
  async run(options: LoadTestOptions): Promise<TestResult> {
    return runLoadTest(options);
  }
}
