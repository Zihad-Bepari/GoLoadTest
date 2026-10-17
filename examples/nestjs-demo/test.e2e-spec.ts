import { GoLoadTestService } from '../../adapters/nestjs/goloadtest.service';

/**
 * Example NestJS E2E Performance / Stress Test
 * Can be run in Jest / Vitest:
 * npm test -- test.e2e-spec.ts
 */
describe('NestJS UsersController Load Test (e2e)', () => {
  const loadTester = new GoLoadTestService();

  it('should maintain P95 latency under 100ms with 30 concurrent users', async () => {
    const result = await loadTester.run({
      name: 'NestJS Users API E2E Benchmark',
      url: 'http://localhost:3000/api/users',
      duration: '5s',
      concurrency: 30,
      thresholds: {
        maxP95: '100ms',
        maxErrorRate: 0.5,
      },
    });

    expect(result.passed).toBe(true);
    expect(result.errorRate).toBeLessThan(0.5);
  }, 30000);
});
