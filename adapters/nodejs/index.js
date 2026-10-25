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
    const tempConfigPath = path.join(os.tmpdir(), `goloadtest-${Date.now()}.json`);
    const tempOutputDir = options.outputPath || path.join(process.cwd(), 'results');

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
      duration: options.duration || '30s',
      concurrency: options.concurrency || 100,
      targetRps: options.targetRps || 0,
      pattern: options.pattern || (options.breakUntilCrash ? 'break-to-crash' : 'constant'),
      stress: options.breakUntilCrash ? {
        enabled: true,
        initialConcurrency: 5,
        stepConcurrency: options.stepConcurrency || 15,
        stepInterval: '2s',
        crashErrorRate: options.crashThreshold || 65.0,
        crashConsecutiveFails: 20,
        crashMaxLatency: '8s'
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
          return resolve(resultData);
        }
      } catch (err) {}

      resolve({ status: 'completed', exitCode: code, rawError: stderrData });
    });
  });
}

// Common REST API route candidates
const defaultCandidateRoutes = [
  { name: 'Health Check', path: '/health', method: 'GET' },
  { name: 'Root API', path: '/api', method: 'GET' },
  { name: 'API v1 Root', path: '/api/v1', method: 'GET' },
  { name: 'Hajj Packages', path: '/api/v1/hajj-packages', method: 'GET' },
  { name: 'Umrah Packages', path: '/api/v1/umrah-packages', method: 'GET' },
  { name: 'Blogs & Articles', path: '/api/v1/blogs', method: 'GET' },
  { name: 'Photo Gallery', path: '/api/v1/gallery', method: 'GET' },
  { name: 'Services', path: '/api/v1/services', method: 'GET' },
  { name: 'FAQs', path: '/api/v1/faqs', method: 'GET' },
  { name: 'Site Config', path: '/api/v1/site-config', method: 'GET' },
  { name: 'Ziyarat Places', path: '/api/v1/ziyarats', method: 'GET' },
  { name: 'Hotels List', path: '/api/v1/hotels', method: 'GET' },
  { name: 'Package Categories', path: '/api/v1/package-categories', method: 'GET' },
  { name: 'Banners & Ads', path: '/api/v1/banners', method: 'GET' },
  { name: 'Why Choose Us', path: '/api/v1/why-choose', method: 'GET' },
  { name: 'Umrah Guidelines', path: '/api/v1/umrah-guidelines', method: 'GET' },
  { name: 'Umrah FAQs', path: '/api/v1/umrah-faqs', method: 'GET' },
  { name: 'Team Members', path: '/api/v1/team-members', method: 'GET' },
  { name: 'Stories', path: '/api/v1/stories', method: 'GET' },
  { name: 'Privacy Policies', path: '/api/v1/privacy-policies', method: 'GET' },
  { name: 'Utilities', path: '/api/v1/utilities', method: 'GET' },
  { name: 'Visas Info', path: '/api/v1/visas', method: 'GET' }
];

/**
 * Express Middleware that directly renders the 1-Click Load Test Landing Page
 * from inside the user's Express server without any iframes or extra ports!
 */
function loadtestMiddleware() {
  let isTesting = false;

  return function (req, res, next) {
    // API endpoint triggered by the "START TEST" button
    if (req.method === 'POST' && (req.path === '/api/start' || req.path === '/start' || req.url.endsWith('/api/start'))) {
      if (isTesting) {
        return res.status(409).json({ error: 'A test is already currently running' });
      }

      isTesting = true;
      const host = req.get('host') || 'localhost:5000';
      const protocol = req.protocol || 'http';
      const baseUrl = `${protocol}://${host}`;

      const endpoints = defaultCandidateRoutes.map(r => ({
        name: r.name,
        url: `${baseUrl}${r.path}`,
        method: r.method,
        timeout: '8s'
      }));

      runLoadTest({
        name: 'Auto 1-Click All API Stress Test',
        endpoints: endpoints,
        duration: '45s',
        concurrency: 500, // Massive concurrency to guarantee breaking point
        breakUntilCrash: true,
        stepConcurrency: 50,
        crashThreshold: 40.0
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
    .container { max-width: 900px; width: 100%; }
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
    th { text-align: left; padding: 0.75rem 1rem; color: var(--muted); font-size: 0.8rem; text-transform: uppercase; border-bottom: 1px solid #1e293b; }
    td { padding: 0.85rem 1rem; border-bottom: 1px solid #1a2233; font-size: 0.9rem; }
    .tag { padding: 0.25rem 0.55rem; border-radius: 6px; font-weight: 700; font-size: 0.75rem; }
    .tag-fast { background: rgba(16, 185, 129, 0.2); color: #34d399; }
    .tag-slow { background: rgba(245, 158, 11, 0.2); color: #fbbf24; }
    .tag-fail { background: rgba(239, 68, 68, 0.2); color: #f87171; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <div class="badge">⚡ GOLOADTEST AUTONOMOUS SUITE</div>
      <h1>Website Auto Stress Tester</h1>
      <p class="subtitle">নিচের বাটনে ক্লিক করলেই স্বয়ংক্রিয়ভাবে আপনার ব্যাকএন্ডের সব API টেস্ট করে সার্ভারের ব্রেকিং পয়েন্ট জানিয়ে দিবে।</p>
    </div>

    <div class="action-card">
      <button id="testBtn" class="big-btn">
        <span class="spinner" id="btnSpinner"></span>
        <span id="btnText">🚀 START ALL API TEST</span>
      </button>
      <div id="statusMsg" class="status-msg">
        ⏳ Go ইঞ্জিন সব API-তে ক্র্যাশ প্রেসার দিচ্ছে... দয়া করে কয়েক সেকেন্ড অপেক্ষা করুন।
      </div>
    </div>

    <div id="resultSection" class="result-section">
      <div class="summary-card">
        <div class="summary-header">
          <h2 style="font-size: 1.3rem;">📊 টেস্টের সহজ ফলাফল (Overview)</h2>
          <span id="resultPill" class="status-pill status-ok">COMPLETED</span>
        </div>

        <div id="verdictText" class="verdict-box good">
          লোড টেস্ট সম্পূর্ণ হয়েছে।
        </div>

        <h3 style="font-size: 1.05rem; margin-top: 1.5rem; margin-bottom: 0.5rem;">🔍 প্রতিটি API-র পারফরম্যান্স রিপোর্ট</h3>
        <table>
          <thead>
            <tr>
              <th>API নাম ও পাথ</th>
              <th>রেসপন্স স্পিড</th>
              <th>স্ট্যাটাস</th>
            </tr>
          </thead>
          <tbody id="endpointTable">
          </tbody>
        </table>
      </div>
    </div>
  </div>

  <script>
    async function startTest() {
      var btn = document.getElementById('testBtn');
      var spinner = document.getElementById('btnSpinner');
      var btnText = document.getElementById('btnText');
      var statusMsg = document.getElementById('statusMsg');
      var resSection = document.getElementById('resultSection');

      btn.disabled = true;
      spinner.style.display = 'inline-block';
      btnText.textContent = 'টেস্টিং চলছে...';
      statusMsg.style.display = 'block';
      resSection.style.display = 'none';

      try {
        var resp = await fetch(window.location.pathname.replace(/\\/?$/, '') + '/api/start', {
          method: 'POST'
        });
        var res = await resp.json();
        renderResult(res);
      } catch (err) {
        alert('টেস্টে সমস্যা হয়েছে: ' + err.message);
      } finally {
        btn.disabled = false;
        spinner.style.display = 'none';
        btnText.textContent = '🚀 START ALL API TEST';
        statusMsg.style.display = 'none';
      }
    }

    function formatLat(ns) {
      if (!ns) return '0ms';
      if (ns > 1000000000) return (ns / 1000000000).toFixed(1) + 's';
      if (ns > 1000000) return Math.round(ns / 1000000) + 'ms';
      return Math.round(ns / 1000) + 'µs';
    }

    function renderResult(res) {
      document.getElementById('resultSection').style.display = 'block';
      var pill = document.getElementById('resultPill');
      var verdict = document.getElementById('verdictText');

      if (res.breakingPoint && res.breakingPoint.detected) {
        pill.className = 'status-pill status-crash';
        pill.textContent = 'SERVER DOWN / BREAKING POINT';
        verdict.className = 'verdict-box bad';
        verdict.innerHTML = '⚠️ <strong>সার্ভার ক্র্যাশ বা ডাউন হয়েছে:</strong> আপনার সার্ভার সর্বোচ্চ <strong>' +
          res.breakingPoint.crashConcurrency + ' জন একসাথে ভিজিটরের</strong> চাপ নিতে পেরেছে। নিরাপদে চালাতে চাইলে কনকারেন্সি <strong>' +
          res.breakingPoint.maxStableConcurrency + ' জনের</strong> মধ্যে রাখা ভালো।<br>কারণ: ' + res.breakingPoint.crashReason;
      } else {
        pill.className = 'status-pill status-ok';
        pill.textContent = 'ALL PASSED & STABLE';
        verdict.className = 'verdict-box good';
        verdict.innerHTML = '🎉 <strong>চমৎকার!</strong> আপনার সার্ভার সব কয়টি API-র সর্বোচ্চ প্রেসার সফলভাবে সামলাতে পেরেছে। কোনো ক্র্যাশ বা অতিরিক্ত স্লোডাউন পাওয়া যায়নি।';
      }

      var tbody = document.getElementById('endpointTable');
      tbody.innerHTML = '';
      (res.endpoints || []).forEach(function(ep) {
        var tr = document.createElement('tr');
        var speedText = formatLat(ep.latency ? ep.latency.median : 0);
        var tagClass = 'tag tag-fast';
        var tagLabel = 'Fast (ভালো)';

        if (ep.errorRate > 5) {
          tagClass = 'tag tag-fail';
          tagLabel = 'Failed (' + ep.errorRate.toFixed(0) + '% Error)';
        } else if (ep.latency && ep.latency.median > 500000000) {
          tagClass = 'tag tag-slow';
          tagLabel = 'Slow (ধীরগতির)';
        }

        tr.innerHTML = '<td><strong>' + ep.name + '</strong><br><small style="color: #64748b;">' + ep.url + '</small></td>' +
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
  resolveBinary
};
