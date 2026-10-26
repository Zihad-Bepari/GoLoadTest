const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

function resolveBinary() {
  const binaryName = process.platform === 'win32' ? 'goloadtest.exe' : 'goloadtest';
  
  const localCandidates = [
    path.resolve(process.cwd(), binaryName),
    path.resolve(process.cwd(), '..', binaryName),
    path.resolve(__dirname, '..', '..', binaryName),
    path.resolve(__dirname, binaryName),
    'C:\\2026\\GoLoadTest\\goloadtest.exe'
  ];

  for (const candidate of localCandidates) {
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }

  return binaryName;
}

/**
 * Runs the GoLoadTest engine with specified options.
 */
function runLoadTest(options) {
  return new Promise((resolve, reject) => {
    const binary = resolveBinary();
    const runId = `glt-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const tempConfigPath = path.join(os.tmpdir(), `goloadtest-${runId}.json`);
    
    // Always use OS temp directory by default so file watchers (nodemon, tsx, watch) NEVER restart the server!
    const tempOutputDir = options.outputPath || path.join(os.tmpdir(), 'goloadtest-runs', runId);

    if (!fs.existsSync(tempOutputDir)) {
      fs.mkdirSync(tempOutputDir, { recursive: true });
    }

    const config = {
      name: options.name || 'API Autonomous Load Test',
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
      duration: options.duration || '25s',
      concurrency: options.concurrency || 200,
      targetRps: options.targetRps || 0,
      pattern: options.pattern || (options.breakUntilCrash ? 'break-to-crash' : 'constant'),
      stress: options.breakUntilCrash ? {
        enabled: true,
        initialConcurrency: options.initialConcurrency || 5,
        stepConcurrency: options.stepConcurrency || 20,
        stepInterval: '2s',
        crashErrorRate: options.crashThreshold || 60.0,
        crashConsecutiveFails: 30,
        crashMaxLatency: '10s'
      } : undefined,
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
    const proc = spawn(binary, args, { stdio: ['ignore', 'pipe', 'pipe'] });

    let stderrData = '';
    proc.stderr.on('data', (d) => { stderrData += d.toString(); });

    proc.on('error', (err) => {
      try { fs.unlinkSync(tempConfigPath); } catch (_) {}
      reject(new Error(`Failed to execute GoLoadTest binary (${binary}): ${err.message}`));
    });

    proc.on('close', (code) => {
      try { fs.unlinkSync(tempConfigPath); } catch (_) {}

      try {
        const files = fs.readdirSync(tempOutputDir)
          .filter(f => f.startsWith('loadtest-') && f.endsWith('.json'))
          .sort()
          .reverse();

        if (files.length > 0) {
          const resultFile = path.join(tempOutputDir, files[0]);
          const resultData = JSON.parse(fs.readFileSync(resultFile, 'utf8'));
          if (!options.outputPath) {
            try { fs.rmSync(tempOutputDir, { recursive: true, force: true }); } catch (_) {}
          }
          return resolve(resultData);
        }
      } catch (err) {}

      resolve({ status: 'completed', exitCode: code, rawError: stderrData });
    });
  });
}

// Helper to extract clean path prefix
function cleanPrefix(p) {
  if (!p) return '';
  let s = String(p).trim();
  if (!s.startsWith('/')) s = '/' + s;
  if (s.endsWith('/') && s.length > 1) s = s.slice(0, -1);
  return s;
}

// Helper to parse express router mount path from its layer regexp
function parseRegexpMount(regexp) {
  if (!regexp || !regexp.source) return '';
  const src = regexp.source;
  if (src === '^\\/?(?=\\/|$)' || src === '^\\/?$' || src === '^\\/') return '';

  const match = src.match(/^\^\\\/([a-zA-Z0-9_\-\/]+?)(?:\\\/\?\(|\/|\$)/);
  if (match && match[1]) {
    return cleanPrefix(match[1].replace(/\\\//g, '/'));
  }

  let cleaned = src
    .replace(/^\^/, '')
    .replace(/\\\/\?\(\?=\\\/\|\$\).*/, '')
    .replace(/\\\/\?\$.*/, '')
    .replace(/\\\//g, '/')
    .replace(/\\/g, '')
    .replace(/\?$/, '');
  if (cleaned === '/' || cleaned === '') return '';
  return cleanPrefix(cleaned);
}

/**
 * Automatically discovers all registered HTTP routes from an Express app or router.
 * Inspects all direct routes, mounted sub-routers, and handles parameterized paths.
 */
function extractExpressRoutes(app) {
  const routes = [];
  if (!app) return routes;

  const router = app._router || (app.router && app.router._router) || (Array.isArray(app.stack) ? app : null);
  if (!router || !Array.isArray(router.stack)) {
    return routes;
  }

  function traverse(stack, basePath) {
    if (!Array.isArray(stack)) return;

    for (const layer of stack) {
      if (!layer) continue;

      if (layer.route) {
        let routePath = layer.route.path;
        if (typeof routePath === 'string') {
          // Normalize parameterized paths: e.g. /users/:id -> /users/1 for reliable testing
          const normalizedPath = routePath.replace(/:[a-zA-Z0-9_]+/g, '1');
          const fullPath = (basePath + cleanPrefix(normalizedPath)).replace(/\/+/g, '/') || '/';

          // Exclude internal loadtest runner paths
          if (fullPath.includes('/loadtest') || fullPath.includes('/api/start') || fullPath.includes('/api/routes') || fullPath.includes('/api/status')) {
            continue;
          }

          const methods = Object.keys(layer.route.methods || {}).filter(m => layer.route.methods[m]);
          for (const method of methods) {
            const m = method.toUpperCase();
            if (['GET', 'POST', 'PUT', 'DELETE', 'PATCH'].includes(m)) {
              routes.push({
                name: `${m} ${fullPath}`,
                path: fullPath,
                method: m
              });
            }
          }
        }
      } else if (layer.name === 'router' && layer.handle && Array.isArray(layer.handle.stack)) {
        const mount = parseRegexpMount(layer.regexp);
        traverse(layer.handle.stack, basePath + mount);
      }
    }
  }

  traverse(router.stack, '');

  // Deduplicate by method + path
  const unique = [];
  const seen = new Set();
  for (const r of routes) {
    const key = `${r.method}:${r.path}`;
    if (!seen.has(key)) {
      seen.add(key);
      unique.push(r);
    }
  }
  return unique;
}

// Fallback REST API route candidates in case app has zero registered routes
const defaultCandidateRoutes = [
  { name: 'Health Check', path: '/health', method: 'GET' },
  { name: 'Root API', path: '/api', method: 'GET' },
  { name: 'API v1 Root', path: '/api/v1', method: 'GET' },
  { name: 'Users List', path: '/api/v1/users', method: 'GET' },
  { name: 'Products List', path: '/api/v1/products', method: 'GET' }
];

/**
 * Runs autonomous all-API stress test by discovering all routes of an Express app.
 */
async function autoTest(app, options = {}) {
  const routes = extractExpressRoutes(app);
  let baseUrl = options.baseUrl;

  if (!baseUrl && app && typeof app.address === 'function') {
    const addr = app.address();
    if (addr && addr.port) {
      baseUrl = `http://127.0.0.1:${addr.port}`;
    }
  }
  if (!baseUrl) {
    baseUrl = 'http://127.0.0.1:3000';
  }

  const selectedRoutes = routes.length > 0 ? routes : defaultCandidateRoutes;
  const endpoints = selectedRoutes.map(r => ({
    name: r.name,
    url: `${baseUrl.replace(/\/$/, '')}${r.path}`,
    method: r.method,
    timeout: options.timeout || '8s'
  }));

  return runLoadTest({
    name: options.name || 'Autonomous All-API Stress Test',
    endpoints: endpoints,
    duration: options.duration || '30s',
    concurrency: options.concurrency || 200,
    breakUntilCrash: options.breakUntilCrash !== false,
    stepConcurrency: options.stepConcurrency || 20,
    crashThreshold: options.crashThreshold || 50.0,
    ...options
  });
}

/**
 * Express Middleware that renders the 1-Click Load Test Landing Page
 * and automatically auto-discovers all project routes and host domain!
 */
function loadtestMiddleware(appOrOptions = {}) {
  let passedApp = null;
  let customOptions = {};

  if (typeof appOrOptions === 'function' || (appOrOptions && appOrOptions._router)) {
    passedApp = appOrOptions;
  } else if (typeof appOrOptions === 'object') {
    passedApp = appOrOptions.app || null;
    customOptions = appOrOptions;
  }

  let isTesting = false;

  return function (req, res, next) {
    const targetApp = passedApp || req.app;
    const rawHost = req.get('host') || '127.0.0.1:5000';
    // Normalize localhost to 127.0.0.1 to prevent Windows IPv6 loopback dropped connections
    const host = rawHost.startsWith('localhost') ? rawHost.replace('localhost', '127.0.0.1') : rawHost;
    const protocol = req.protocol || 'http';
    const baseUrl = customOptions.baseUrl || `${protocol}://${host}`;
    const reqPath = req.path || '';
    const reqUrl = req.url || '';

    // API endpoint returning list of auto-discovered routes
    if (req.method === 'GET' && (reqPath === '/api/routes' || reqPath.endsWith('/api/routes') || reqUrl.endsWith('/api/routes'))) {
      const discovered = extractExpressRoutes(targetApp);
      const routes = discovered.length > 0 ? discovered : defaultCandidateRoutes;
      return res.json({
        baseUrl: baseUrl,
        totalRoutes: routes.length,
        routes: routes
      });
    }

    // API endpoint triggered by the "START TEST" button
    if (req.method === 'POST' && (reqPath === '/api/start' || reqPath === '/start' || reqPath.endsWith('/api/start') || reqUrl.endsWith('/api/start'))) {
      if (isTesting) {
        return res.status(409).json({ error: 'A test is already currently running' });
      }

      isTesting = true;
      const discovered = extractExpressRoutes(targetApp);
      const routesToTest = discovered.length > 0 ? discovered : defaultCandidateRoutes;

      const endpoints = routesToTest.map(r => ({
        name: r.name,
        url: `${baseUrl}${r.path}`,
        method: r.method,
        timeout: customOptions.timeout || '8s'
      }));

      runLoadTest({
        name: customOptions.name || 'Autonomous All-API Stress Test',
        endpoints: endpoints,
        duration: customOptions.duration || '15m', // Open-ended duration: runs until actual server breakdown!
        concurrency: customOptions.concurrency || 5000, // Massive 5,000 user ramp ceiling!
        breakUntilCrash: true,
        initialConcurrency: 10,
        stepConcurrency: customOptions.stepConcurrency || 35,
        crashThreshold: customOptions.crashThreshold || 60.0
      })
      .then((result) => {
        isTesting = false;
        res.json(result);
      })
      .catch((err) => {
        isTesting = false;
        res.status(500).json({ error: err.message });
      });

      return;
    }

    // Direct HTML Landing Page (Zero iframe, 100% native)
    res.setHeader('Content-Type', 'text/html; charset=utf-8');
    res.send(`<!DOCTYPE html>
<html lang="bn">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>GoLoadTest · 1-Click All API Stress Tester</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;600;700;800&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #090d16;
      --card: #111726;
      --border: #1e293b;
      --primary: #3b82f6;
      --success: #10b981;
      --danger: #ef4444;
      --warning: #f59e0b;
      --text: #f8fafc;
      --muted: #94a3b8;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: var(--bg);
      color: var(--text);
      font-family: 'Plus Jakarta Sans', sans-serif;
      padding: 2.5rem 1rem;
      display: flex;
      justify-content: center;
    }
    .container { max-width: 1150px; width: 100%; }
    .header { text-align: center; margin-bottom: 2rem; }
    .badge {
      display: inline-block;
      background: rgba(59, 130, 246, 0.15);
      color: #60a5fa;
      padding: 0.35rem 0.9rem;
      border-radius: 9999px;
      font-size: 0.85rem;
      font-weight: 700;
      letter-spacing: 0.5px;
      margin-bottom: 0.75rem;
      border: 1px solid rgba(59, 130, 246, 0.3);
    }
    h1 { font-size: 2.2rem; font-weight: 800; color: #fff; margin-bottom: 0.5rem; }
    p.subtitle { color: var(--muted); font-size: 1rem; }

    .action-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 20px;
      padding: 3rem 2rem;
      text-align: center;
      box-shadow: 0 10px 40px rgba(0,0,0,0.4);
      margin-bottom: 2rem;
    }
    .big-btn {
      background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
      color: #fff;
      font-size: 1.3rem;
      font-weight: 800;
      padding: 1.2rem 3rem;
      border-radius: 14px;
      border: none;
      cursor: pointer;
      box-shadow: 0 6px 24px rgba(37, 99, 235, 0.5);
      transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
      display: inline-flex;
      align-items: center;
      gap: 0.75rem;
    }
    .big-btn:hover { transform: translateY(-2px); box-shadow: 0 10px 32px rgba(37, 99, 235, 0.7); }
    .big-btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none; }

    .spinner {
      display: none;
      width: 24px;
      height: 24px;
      border: 3px solid rgba(255,255,255,0.3);
      border-radius: 50%;
      border-top-color: #fff;
      animation: spin 0.8s ease-in-out infinite;
    }
    @keyframes spin { to { transform: rotate(360deg); } }

    .status-msg {
      display: none;
      margin-top: 1.5rem;
      font-size: 1.05rem;
      font-weight: 600;
      color: #93c5fd;
    }

    .result-section { display: none; }
    .summary-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 16px;
      padding: 1.75rem;
      margin-bottom: 1.5rem;
    }
    .summary-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 1.25rem;
    }
    .status-pill {
      padding: 0.4rem 0.9rem;
      border-radius: 8px;
      font-weight: 800;
      font-size: 0.85rem;
    }
    .status-ok { background: rgba(16, 185, 129, 0.2); color: #34d399; }
    .status-crash { background: rgba(239, 68, 68, 0.2); color: #f87171; }

    /* Executive Short Summary Grid */
    .summary-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 1rem;
      margin-bottom: 1.5rem;
    }
    .metric-card {
      background: rgba(255, 255, 255, 0.03);
      border: 1px solid #1e293b;
      border-radius: 14px;
      padding: 1.25rem 1rem;
      text-align: center;
      transition: transform 0.2s, border-color 0.2s;
    }
    .metric-card:hover {
      transform: translateY(-2px);
      border-color: rgba(59, 130, 246, 0.5);
    }
    .metric-title {
      font-size: 0.82rem;
      font-weight: 700;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.5px;
      margin-bottom: 0.45rem;
    }
    .metric-val {
      font-size: 1.65rem;
      font-weight: 800;
      color: #fff;
      line-height: 1.2;
    }
    .metric-val.crash { color: #f87171; }
    .metric-val.safe { color: #34d399; }
    .metric-val.rps { color: #60a5fa; }
    .metric-sub {
      font-size: 0.78rem;
      color: #94a3b8;
      margin-top: 0.35rem;
    }

    .verdict-box {
      padding: 1.1rem 1.4rem;
      border-radius: 12px;
      margin-bottom: 1.5rem;
      line-height: 1.6;
      font-size: 1rem;
    }
    .verdict-box.good { background: rgba(16, 185, 129, 0.1); border-left: 4px solid var(--success); }
    .verdict-box.bad { background: rgba(239, 68, 68, 0.1); border-left: 4px solid var(--danger); }

    table { width: 100%; border-collapse: collapse; margin-top: 1rem; }
    th { text-align: left; padding: 0.85rem 0.75rem; color: var(--muted); font-size: 0.8rem; text-transform: uppercase; border-bottom: 1px solid #1e293b; }
    td { padding: 0.85rem 0.75rem; border-bottom: 1px solid #1a2233; font-size: 0.9rem; vertical-align: middle; }
    .tag { padding: 0.35rem 0.65rem; border-radius: 6px; font-weight: 700; font-size: 0.75rem; display: inline-block; }
    .tag-ok { background: rgba(16, 185, 129, 0.2); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.4); }
    .tag-slow { background: rgba(245, 158, 11, 0.2); color: #fbbf24; border: 1px solid rgba(245, 158, 11, 0.4); }
    .tag-degraded { background: rgba(245, 158, 11, 0.25); color: #f59e0b; border: 1px solid rgba(245, 158, 11, 0.5); }
    .tag-crash { background: rgba(239, 68, 68, 0.25); color: #f87171; border: 1px solid rgba(239, 68, 68, 0.5); }
    .tag-offline { background: rgba(148, 163, 184, 0.15); color: #94a3b8; border: 1px solid rgba(148, 163, 184, 0.3); }

    .scanning-box {
      background: linear-gradient(180deg, #111a2e 0%, #0c1220 100%);
      border: 1px solid #2563eb;
      box-shadow: 0 0 30px rgba(37, 99, 235, 0.25);
      border-radius: 18px;
      padding: 2.25rem 2rem;
      margin-top: 1.5rem;
      text-align: center;
      animation: pulseGlow 2s infinite alternate;
    }
    @keyframes pulseGlow {
      from { box-shadow: 0 0 15px rgba(37, 99, 235, 0.2); }
      to { box-shadow: 0 0 35px rgba(37, 99, 235, 0.45); }
    }
    .radar {
      width: 56px;
      height: 56px;
      margin: 0 auto 1.25rem;
      border: 3px solid rgba(59, 130, 246, 0.3);
      border-radius: 50%;
      position: relative;
    }
    .radar::after {
      content: '';
      position: absolute;
      top: -3px; left: -3px; right: -3px; bottom: -3px;
      border: 3px solid transparent;
      border-top-color: #60a5fa;
      border-radius: 50%;
      animation: spin 0.9s linear infinite;
    }
    .scan-title { font-size: 1.35rem; font-weight: 800; color: #fff; margin-bottom: 0.4rem; }
    .scan-sub { color: #94a3b8; font-size: 0.95rem; margin-bottom: 1.25rem; }
    .scan-bar-bg { background: #1e293b; height: 10px; border-radius: 9999px; overflow: hidden; max-width: 480px; margin: 0 auto 1.25rem; }
    .scan-bar-fill { background: linear-gradient(90deg, #3b82f6, #60a5fa, #10b981); height: 100%; width: 30%; animation: indeterminate 2s infinite ease-in-out; }
    @keyframes indeterminate {
      0% { width: 10%; margin-left: 0; }
      50% { width: 55%; margin-left: 20%; }
      100% { width: 15%; margin-left: 85%; }
    }
    .scan-metrics { display: flex; justify-content: center; gap: 2.5rem; font-size: 0.95rem; font-weight: 600; color: #cbd5e1; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <div class="badge">⚡ GOLOADTEST AUTONOMOUS SUITE</div>
      <h1>Website Auto Stress &amp; Capacity Tester</h1>
      <p class="subtitle">নিচের বাটনে ক্লিক করলেই স্বয়ংক্রিয়ভাবে সব API-তে ধাপে ধাপে ট্রাফিক বাড়িয়ে সার্ভারের আসল ব্রেকিং পয়েন্ট ও ইউজার ধারণক্ষমতা জানিয়ে দিবে।</p>
    </div>

    <div class="action-card">
      <button id="testBtn" class="big-btn">
        <span class="spinner" id="btnSpinner"></span>
        <span id="btnText">🚀 START ALL API STRESS TEST</span>
      </button>

      <!-- Live Scanning Console -->
      <div id="scanningBox" class="scanning-box" style="display: none;">
        <div class="radar"></div>
        <div class="scan-title">🔍 Scanning &amp; Stress Testing All APIs...</div>
        <div id="scanSub" class="scan-sub">সার্ভারের সব কয়টি এন্ডপয়েন্টে কনকারেন্সি বাড়িয়ে স্ট্রেস টেস্ট চলছে...</div>
        <div class="scan-bar-bg">
          <div class="scan-bar-fill"></div>
        </div>
        <div class="scan-metrics">
          <span id="scanElapsed">⏱️ সময়: 0s</span>
          <span id="scanStatus">⚡ প্রেসার বাড়ছে (Ramping Up)...</span>
        </div>
      </div>
    </div>

    <div id="resultSection" class="result-section">
      <div class="summary-card">
        <div class="summary-header">
          <h2 style="font-size: 1.3rem;">📊 টেস্টের ফলাফল ও ব্রেকিং পয়েন্ট সামারি</h2>
          <span id="resultPill" class="status-pill status-ok">COMPLETED</span>
        </div>

        <!-- 4-Card Executive Short Summary Grid -->
        <div class="summary-grid">
          <div class="metric-card">
            <div class="metric-title">💥 ব্রেকিং পয়েন্ট</div>
            <div class="metric-val" id="sumCrashCon">-</div>
            <div class="metric-sub" id="sumCrashSub">যে চাপে সার্ভার ফেইল করে</div>
          </div>
          <div class="metric-card">
            <div class="metric-title">🛡️ নিরাপদ ধারণক্ষমতা</div>
            <div class="metric-val safe" id="sumStableCon">-</div>
            <div class="metric-sub" id="sumStableSub">সর্বোচ্চ টেকসই ইউজার</div>
          </div>
          <div class="metric-card">
            <div class="metric-title">🚀 পিক থ্রুপুট</div>
            <div class="metric-val rps" id="sumRps">-</div>
            <div class="metric-sub" id="sumRpsSub">প্রতি সেকেন্ডে রেসপন্স</div>
          </div>
          <div class="metric-card">
            <div class="metric-title">📈 মোট রিকোয়েস্ট ও সময়</div>
            <div class="metric-val" id="sumTotal">-</div>
            <div class="metric-sub" id="sumDuration">রান টাইম</div>
          </div>
        </div>

        <div id="verdictText" class="verdict-box good">
          লোড টেস্ট সম্পূর্ণ হয়েছে।
        </div>

        <h3 style="font-size: 1.05rem; margin-top: 1.75rem; margin-bottom: 0.75rem;">🔍 প্রতিটি API-র আলাদা ক্যাপাসিটি ও পারফরম্যান্স রিপোর্ট</h3>
        <div style="overflow-x: auto;">
          <table>
            <thead>
              <tr>
                <th>API নাম ও মেথড</th>
                <th>হ্যান্ডেল করা রিকোয়েস্ট</th>
                <th>ইউজার ধারণক্ষমতা (Capacity)</th>
                <th>স্পিড (RPS)</th>
                <th>ল্যাটেন্সি (Latency)</th>
                <th>ব্রেকিং পয়েন্ট ও স্ট্যাটাস</th>
              </tr>
            </thead>
            <tbody id="endpointTable">
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>

  <script>
    async function startTest() {
      var btn = document.getElementById('testBtn');
      var spinner = document.getElementById('btnSpinner');
      var btnText = document.getElementById('btnText');
      var scanBox = document.getElementById('scanningBox');
      var resSection = document.getElementById('resultSection');

      btn.disabled = true;
      spinner.style.display = 'inline-block';
      btnText.textContent = 'টেস্টিং চলছে...';
      scanBox.style.display = 'block';
      resSection.style.display = 'none';

      var elapsedSec = 0;
      var timerId = setInterval(function() {
        elapsedSec++;
        document.getElementById('scanElapsed').textContent = '⏱️ সময়: ' + elapsedSec + 's';
        var estUsers = 10 + Math.floor(elapsedSec / 2) * 35;
        document.getElementById('scanStatus').textContent = '👥 ট্রাফিক বাড়ছে: ~' + estUsers + ' জন ইউজার...';
      }, 1000);

      try {
        var basePath = window.location.pathname;
        while (basePath.length > 1 && basePath.charAt(basePath.length - 1) === '/') {
          basePath = basePath.slice(0, -1);
        }
        var startUrl = (basePath === '/' ? '' : basePath) + '/api/start';

        var resp = await fetch(startUrl, {
          method: 'POST'
        });
        var res = await resp.json();
        renderResult(res, elapsedSec);
      } catch (err) {
        alert('টেস্টে সমস্যা হয়েছে: ' + err.message);
      } finally {
        clearInterval(timerId);
        btn.disabled = false;
        spinner.style.display = 'none';
        btnText.textContent = '🚀 START ALL API STRESS TEST';
        scanBox.style.display = 'none';
      }
    }

    function formatLat(ns) {
      if (!ns) return '0ms';
      if (ns > 1000000000) return (ns / 1000000000).toFixed(1) + 's';
      if (ns > 1000000) return Math.round(ns / 1000000) + 'ms';
      return Math.round(ns / 1000) + 'µs';
    }

    function renderResult(res, elapsedSec) {
      document.getElementById('resultSection').style.display = 'block';
      var pill = document.getElementById('resultPill');
      var verdict = document.getElementById('verdictText');

      var crashEl = document.getElementById('sumCrashCon');
      var crashSubEl = document.getElementById('sumCrashSub');
      var stableEl = document.getElementById('sumStableCon');
      var stableSubEl = document.getElementById('sumStableSub');
      var rpsEl = document.getElementById('sumRps');
      var totalEl = document.getElementById('sumTotal');
      var durEl = document.getElementById('sumDuration');

      var totalReqs = res.totalRequests || 0;
      var throughput = Math.round(res.throughputRps || 0);
      var durSec = (res.duration && res.duration > 0) ? (res.duration / 1000000000).toFixed(1) : (elapsedSec || 1);

      totalEl.textContent = totalReqs.toLocaleString('bn-BD') + ' টি';
      durEl.textContent = 'মোট সময়: ' + durSec + 's';
      rpsEl.textContent = throughput.toLocaleString('bn-BD') + ' RPS';

      var isOffline = (totalReqs === 0) || (res.abortReason && (res.abortReason.includes('offline') || res.abortReason.includes('refused')));

      if (isOffline) {
        pill.className = 'status-pill status-crash';
        pill.textContent = 'SERVER OFFLINE / UNREACHABLE';
        verdict.className = 'verdict-box bad';
        verdict.innerHTML = '🚨 <strong>টার্গেট সার্ভারটি চালু নেই বা অফলাইন:</strong> GoLoadTest সার্ভারে কানেক্ট করতে পারেনি (' + (res.abortReason || 'Connection Refused') + ')।<br><br>' +
          '👉 <strong>করণীয়:</strong> টার্মিনালে আপনার ব্যাকএন্ড সার্ভারটি চালু রাখুন (যেমন: <code>npm run dev</code> বা <code>node src/server.js</code>), এবং সার্ভার চালু হওয়ার পর পুনরায় এই টেস্টটি চালান।';

        crashEl.textContent = 'অফলাইন';
        crashEl.className = 'metric-val crash';
        crashSubEl.textContent = 'সার্ভারে কানেক্ট করা যায়নি';
        stableEl.textContent = '০ ইউজার';
        stableEl.className = 'metric-val crash';
        stableSubEl.textContent = 'সার্ভার বন্ধ ছিল';
      } else if (res.breakingPoint && res.breakingPoint.detected) {
        pill.className = 'status-pill status-crash';
        pill.textContent = 'BREAKING POINT IDENTIFIED';
        verdict.className = 'verdict-box bad';
        verdict.innerHTML = '⚠️ <strong>সার্ভারের ব্রেকিং পয়েন্ট পাওয়া গেছে:</strong> আপনার সার্ভার সর্বোচ্চ <strong>' +
          (res.breakingPoint.maxStableConcurrency || 'N/A') + ' জন ইউজারের</strong> চাপ নিরাপদে সামলাতে পেরেছে। এরপর <strong>' +
          res.breakingPoint.crashConcurrency + ' জন ইউজারের</strong> চাপে সার্ভার ক্র্যাশ বা ফেইল করেছে।<br>' +
          '📊 সর্বোচ্চ সুরক্ষিত স্পিড: <strong>' + Math.round(res.breakingPoint.maxStableRps || res.throughputRps) + ' RPS</strong> | কারণ: ' + res.breakingPoint.crashReason;

        crashEl.textContent = res.breakingPoint.crashConcurrency + ' ইউজার';
        crashEl.className = 'metric-val crash';
        crashSubEl.textContent = 'এই চাপে ড্রপ / ফেইল হয়েছে';
        stableEl.textContent = (res.breakingPoint.maxStableConcurrency || '১') + ' ইউজার';
        stableEl.className = 'metric-val safe';
        stableSubEl.textContent = 'সর্বোচ্চ সুরক্ষিত লোড';
      } else {
        pill.className = 'status-pill status-ok';
        pill.textContent = 'ALL PASSED & STABLE';
        verdict.className = 'verdict-box good';
        verdict.innerHTML = '🎉 <strong>চমৎকার!</strong> টেস্ট চলাকালীন সব কয়টি API সফলভাবে প্রেসার সামলেছে। মোট রিকোয়েস্ট: <strong>' + totalReqs.toLocaleString('bn-BD') + '</strong> | গড় থ্রুপুট: <strong>' + throughput.toLocaleString('bn-BD') + ' RPS</strong>।';

        var maxObserved = 0;
        (res.endpoints || []).forEach(function(ep) {
          if (ep.maxUsers && ep.maxUsers > maxObserved) maxObserved = ep.maxUsers;
        });

        crashEl.textContent = 'ক্র্যাশ করেনি';
        crashEl.className = 'metric-val safe';
        crashSubEl.textContent = '>' + (maxObserved || 1000) + ' ইউজারেও অক্ষত';
        stableEl.textContent = (maxObserved || 1000) + '+ ইউজার';
        stableEl.className = 'metric-val safe';
        stableSubEl.textContent = 'সব কয়টি API সফল';
      }

      var tbody = document.getElementById('endpointTable');
      tbody.innerHTML = '';
      (res.endpoints || []).forEach(function(ep) {
        var tr = document.createElement('tr');
        var total = ep.totalRequests || 0;
        var success = ep.successRequests || 0;
        var failed = ep.failedRequests || 0;
        var users = ep.maxUsers || 0;
        var breakingUsers = ep.breakingUsers || 0;
        var rps = ep.rps ? Math.round(ep.rps) : 0;
        var speedText = total > 0 ? formatLat(ep.latency ? (ep.latency.mean || ep.latency.median) : 0) : '-';

        var reqDisplay = total > 0 ? (success + ' সফল / ' + total + ' মোট') : '<span style="color:#94a3b8;">০ রিকোয়েস্ট</span>';
        var userDisplay = '-';
        var tagClass = 'tag tag-ok';
        var tagLabel = ep.statusLabel || 'Stable';

        if (total === 0) {
          tagClass = 'tag tag-offline';
          tagLabel = '⚠️ Not Reached (সার্ভার অফলাইন)';
          userDisplay = '<span style="color:#94a3b8;">-</span>';
          reqDisplay = '<span style="color:#ef4444;">০ / ০ (সার্ভার ডাউন ছিল)</span>';
        } else if (failed > 0 && failed >= total * 0.8) {
          tagClass = 'tag tag-crash';
          tagLabel = '💥 Crashed (' + ep.errorRate.toFixed(0) + '% Error)';
          userDisplay = breakingUsers ? ('ফেইল করেছে: ' + breakingUsers + ' ইউজারে') : ('ক্র্যাশ করেছে');
        } else if (failed > 0) {
          tagClass = 'tag tag-degraded';
          tagLabel = '⚠️ Degraded (' + ep.errorRate.toFixed(0) + '% Error)';
          userDisplay = users ? ('টিকে ছিল: ' + users + ' ইউজার') : '-';
        } else {
          tagClass = 'tag tag-ok';
          tagLabel = '✅ Stable (হ্যান্ডেল করেছে)';
          userDisplay = users ? (users + ' ইউজার হ্যান্ডেল করেছে') : (res.breakingPoint && res.breakingPoint.maxStableConcurrency ? res.breakingPoint.maxStableConcurrency + ' ইউজার' : '১০০+ ইউজার');
        }

        tr.innerHTML = '<td><strong>' + ep.name + '</strong><br><small style="color: #64748b;">' + ep.url + '</small></td>' +
          '<td>' + reqDisplay + '</td>' +
          '<td><strong>' + userDisplay + '</strong></td>' +
          '<td>' + (total > 0 ? (rps + ' req/s') : '-') + '</td>' +
          '<td>' + speedText + '</td>' +
          '<td><span class="' + tagClass + '">' + tagLabel + '</span></td>';
        tbody.appendChild(tr);
      });
    }

    document.getElementById('testBtn').addEventListener('click', startTest);
  </script>
</body>
</html>`);
  };
}

module.exports = {
  loadtestMiddleware,
  runLoadTest,
  resolveBinary,
  autoTest,
  extractRoutes: extractExpressRoutes
};
