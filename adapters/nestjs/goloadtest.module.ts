import { Module } from '@nestjs/common';
import { GoLoadTestService } from './goloadtest.service';

/**
 * GoLoadTestModule — NestJS module for high-performance API load testing.
 *
 * Import this module into your AppModule (or a specific feature module / e2e test module)
 * to make GoLoadTestService available via dependency injection.
 *
 * Usage in AppModule:
 * ```typescript
 * @Module({
 *   imports: [GoLoadTestModule],
 * })
 * export class AppModule {}
 * ```
 *
 * Usage in an e2e test:
 * ```typescript
 * const moduleRef = await Test.createTestingModule({
 *   imports: [GoLoadTestModule],
 * }).compile();
 *
 * const loadTestService = moduleRef.get(GoLoadTestService);
 * const result = await loadTestService.run({
 *   url: 'http://localhost:3000/api/users',
 *   duration: '30s',
 *   concurrency: 50,
 * });
 * ```
 */
@Module({
  providers: [GoLoadTestService],
  exports: [GoLoadTestService],
})
export class GoLoadTestModule {}
