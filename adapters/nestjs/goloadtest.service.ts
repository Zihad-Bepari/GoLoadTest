import { runLoadTest, LoadTestOptions, TestResult } from '../nodejs';

/**
 * GoLoadTestService allows testing NestJS controllers and services directly
 * in integration tests (e.g. e2e tests with Jest or supertest).
 */
export class GoLoadTestService {
  /**
   * Runs load testing against a running NestJS application endpoint.
   */
  async run(options: LoadTestOptions): Promise<TestResult> {
    return runLoadTest(options);
  }
}
