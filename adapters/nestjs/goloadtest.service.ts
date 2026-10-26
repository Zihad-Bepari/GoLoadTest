import { Injectable } from '@nestjs/common';
import { runLoadTest, autoTest, extractRoutes } from '../nodejs';
import type { LoadTestOptions, TestResult, DiscoveredRoute } from '../nodejs';

/**
 * GoLoadTestService — NestJS injectable service for high-performance load testing.
 *
 * Usage in a module:
 *   Import GoLoadTestModule into any NestJS module, then inject GoLoadTestService
 *   via constructor injection in your test class or service.
 *
 * Example (Auto all-routes discovery):
 *   const result = await this.loadTestService.autoTest(app, {
 *     duration: '20s',
 *     breakUntilCrash: true,
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

  /**
   * Automatically discovers all routes from the NestJS application instance,
   * resolves its base URL, and executes a full stress test until breaking point.
   */
  async autoTest(app: any, options: LoadTestOptions = {}): Promise<TestResult> {
    let baseUrl = options.baseUrl;
    if (!baseUrl && typeof app.getUrl === 'function') {
      try {
        baseUrl = await app.getUrl();
        baseUrl = baseUrl.replace('[::1]', '127.0.0.1').replace('0.0.0.0', '127.0.0.1');
      } catch (_) {}
    }

    const instance = (app.getHttpAdapter && app.getHttpAdapter())
      ? app.getHttpAdapter().getInstance()
      : (app.getHttpServer ? app.getHttpServer() : app);

    return autoTest(instance, { ...options, baseUrl });
  }

  /**
   * Discovers all routes from a NestJS application.
   */
  extractRoutes(app: any): DiscoveredRoute[] {
    const instance = (app.getHttpAdapter && app.getHttpAdapter())
      ? app.getHttpAdapter().getInstance()
      : (app.getHttpServer ? app.getHttpServer() : app);
    return extractRoutes(instance);
  }
}
