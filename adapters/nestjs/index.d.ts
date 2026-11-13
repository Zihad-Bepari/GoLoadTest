import { LoadTestOptions, TestResult, DiscoveredRoute } from '../nodejs';

export declare class GoLoadTestService {
  run(options: LoadTestOptions): Promise<TestResult>;
  autoTest(app: any, options?: LoadTestOptions): Promise<TestResult>;
  extractRoutes(app: any): DiscoveredRoute[];
}

export declare class GoLoadTestModule {}
