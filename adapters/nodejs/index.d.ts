export interface EndpointConfig {
  name: string;
  url: string;
  method?: string;
  headers?: Record<string, string>;
  body?: string | Record<string, any>;
  timeout?: string;
  expectedStatus?: number[];
  weight?: number;
}

export interface ThresholdConfig {
  maxP95?: string;
  maxP99?: string;
  maxAvgLatency?: string;
  maxErrorRate?: number;
  minThroughput?: number;
}

export interface LoadTestOptions {
  name?: string;
  url?: string;
  method?: string;
  headers?: Record<string, string>;
  body?: string | Record<string, any>;
  endpoints?: EndpointConfig[];
  duration?: string;
  concurrency?: number;
  targetRps?: number;
  pattern?: 'constant' | 'ramp' | 'spike' | 'step';
  thresholds?: ThresholdConfig;
  circuitBreaker?: {
    enabled?: boolean;
    maxErrorRate?: number;
    consecutiveFails?: number;
  };
  outputPath?: string;
  maxP95?: string;
  maxErrorRate?: number;
}

export interface TestResult {
  suiteName: string;
  totalRequests: number;
  successRequests: number;
  failedRequests: number;
  errorRate: number;
  throughputRps: number;
  latency: {
    min: number;
    max: number;
    mean: number;
    median: number;
    p90: number;
    p95: number;
    p99: number;
  };
  passed: boolean;
  aborted: boolean;
  abortReason?: string;
  bottlenecks?: Array<{
    title: string;
    severity: string;
    description: string;
    evidence: string;
    suggestion: string;
  }>;
}

export function runLoadTest(options: LoadTestOptions): Promise<TestResult>;
export function resolveBinary(): string;
