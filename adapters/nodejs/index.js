const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

function resolveBinary() {
  const binaryName = process.platform === 'win32' ? 'goloadtest.exe' : 'goloadtest';
  
  // 1. Check current directory or parent directory
  const localCandidates = [
    path.resolve(process.cwd(), binaryName),
    path.resolve(process.cwd(), '..', binaryName),
    path.resolve(__dirname, '..', '..', binaryName),
    path.resolve(__dirname, binaryName)
  ];

  for (const candidate of localCandidates) {
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }

  // 2. Fall back to binary on PATH
  return binaryName;
}

/**
 * Runs a high-performance load test using GoLoadTest core engine.
 * @param {Object} options Configuration options
 * @returns {Promise<Object>} The parsed TestResult
 */
function runLoadTest(options) {
  return new Promise((resolve, reject) => {
    const binary = resolveBinary();

    // Prepare temp config file
    const tempConfigPath = path.join(os.tmpdir(), `goloadtest-${Date.now()}.json`);
    const tempOutputDir = options.outputPath || path.join(process.cwd(), 'results');

    const config = {
      name: options.name || 'Node.js API Load Test',
      endpoints: options.endpoints || [
        {
          name: options.name || 'Default Endpoint',
          url: options.url,
          method: options.method || 'GET',
          headers: options.headers || {},
          body: typeof options.body === 'object' ? JSON.stringify(options.body) : (options.body || ''),
          timeout: options.timeout || '10s',
          expectedStatus: options.expectedStatus || [200]
        }
      ],
      duration: options.duration || '10s',
      concurrency: options.concurrency || 10,
      targetRps: options.targetRps || 0,
      pattern: options.pattern || 'constant',
      thresholds: options.thresholds || {
        maxP95: options.maxP95 || '300ms',
        maxErrorRate: options.maxErrorRate !== undefined ? options.maxErrorRate : 2.0
      },
      circuitBreaker: options.circuitBreaker || {
        enabled: true,
        maxErrorRate: 50.0
      },
      monitor: options.monitor || { enabled: true, localHost: true },
      outputFormat: 'json',
      outputPath: tempOutputDir
    };

    fs.writeFileSync(tempConfigPath, JSON.stringify(config, null, 2));

    const args = ['run', '--config', tempConfigPath, '--output', tempOutputDir, '--format', 'json'];
    const proc = spawn(binary, args, { stdio: 'inherit' });

    proc.on('error', (err) => {
      try { fs.unlinkSync(tempConfigPath); } catch (_) {}
      reject(new Error(`Failed to execute GoLoadTest binary (${binary}): ${err.message}`));
    });

    proc.on('close', (code) => {
      try { fs.unlinkSync(tempConfigPath); } catch (_) {}

      // Find the latest generated result json
      try {
        const files = fs.readdirSync(tempOutputDir)
          .filter(f => f.startsWith('loadtest-') && f.endsWith('.json'))
          .sort()
          .reverse();

        if (files.length > 0) {
          const resultFile = path.join(tempOutputDir, files[0]);
          const resultData = JSON.parse(fs.readFileSync(resultFile, 'utf8'));
          
          if (code !== 0) {
            resultData.exitCode = code;
          }
          return resolve(resultData);
        }
      } catch (err) {
        // Fallback
      }

      if (code !== 0) {
        return reject(new Error(`GoLoadTest exited with status code ${code} (SLA Thresholds breached or aborted)`));
      }
      resolve({ status: 'completed', exitCode: code });
    });
  });
}

module.exports = {
  runLoadTest,
  resolveBinary
};
