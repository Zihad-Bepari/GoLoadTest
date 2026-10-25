package server

func getDashboardHTML() string {
	return `<!DOCTYPE html>
<html lang="bn">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>GoLoadTest · 1-Click All API Stress Tester</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;600;700;800&family=JetBrains+Mono:wght@500;700&display=swap" rel="stylesheet">
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
      padding: 2rem 1rem;
      display: flex;
      justify-content: center;
    }
    .container { max-width: 900px; width: 100%; }
    
    /* Top Header */
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

    /* Main Big Action Card */
    .action-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 20px;
      padding: 2.5rem 2rem;
      text-align: center;
      box-shadow: 0 10px 40px rgba(0,0,0,0.4);
      margin-bottom: 2rem;
      position: relative;
      overflow: hidden;
    }
    .action-card::before {
      content: "";
      position: absolute;
      top: -50%;
      left: 50%;
      transform: translateX(-50%);
      width: 400px;
      height: 200px;
      background: radial-gradient(ellipse, rgba(59, 130, 246, 0.15), transparent 70%);
      pointer-events: none;
    }

    .input-group {
      max-width: 500px;
      margin: 0 auto 1.5rem auto;
      display: flex;
      gap: 0.5rem;
      background: #182234;
      padding: 0.4rem;
      border-radius: 12px;
      border: 1px solid #334155;
    }
    .input-group input {
      flex: 1;
      background: transparent;
      border: none;
      outline: none;
      color: #fff;
      padding: 0.5rem 1rem;
      font-size: 1rem;
      font-family: inherit;
    }

    /* Giant 1-Click Button */
    .big-btn {
      background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
      color: #fff;
      font-size: 1.25rem;
      font-weight: 800;
      padding: 1.1rem 2.8rem;
      border-radius: 14px;
      border: none;
      cursor: pointer;
      box-shadow: 0 4px 20px rgba(37, 99, 235, 0.5);
      transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
      display: inline-flex;
      align-items: center;
      gap: 0.75rem;
    }
    .big-btn:hover { transform: translateY(-2px); box-shadow: 0 8px 30px rgba(37, 99, 235, 0.7); }
    .big-btn:active { transform: translateY(0); }
    .big-btn.running {
      background: linear-gradient(135deg, #ef4444 0%, #b91c1c 100%);
      box-shadow: 0 4px 20px rgba(239, 68, 68, 0.5);
    }

    /* Live Progress Bar */
    .progress-box {
      display: none;
      margin-top: 1.5rem;
      background: #141c2e;
      border: 1px solid #23314d;
      border-radius: 12px;
      padding: 1.25rem;
      text-align: left;
    }
    .progress-title {
      font-weight: 700;
      color: #60a5fa;
      display: flex;
      justify-content: space-between;
      margin-bottom: 0.75rem;
      font-size: 0.95rem;
    }
    .live-stats-bar {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 0.75rem;
    }
    .mini-stat {
      background: #0d1320;
      padding: 0.75rem;
      border-radius: 8px;
      text-align: center;
    }
    .mini-label { font-size: 0.75rem; color: var(--muted); }
    .mini-val { font-size: 1.25rem; font-weight: 800; color: #fff; margin-top: 0.2rem; }

    /* Results Card */
    .result-section { display: none; }
    .summary-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 16px;
      padding: 1.5rem;
      margin-bottom: 1.5rem;
    }
    .summary-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 1rem;
    }
    .status-pill {
      padding: 0.4rem 0.8rem;
      border-radius: 8px;
      font-weight: 800;
      font-size: 0.85rem;
    }
    .status-ok { background: rgba(16, 185, 129, 0.2); color: #34d399; }
    .status-crash { background: rgba(239, 68, 68, 0.2); color: #f87171; }

    /* Human-friendly Insights */
    .verdict-box {
      padding: 1rem 1.25rem;
      border-radius: 12px;
      margin-bottom: 1.5rem;
      line-height: 1.6;
      font-size: 0.95rem;
    }
    .verdict-box.good { background: rgba(16, 185, 129, 0.1); border-left: 4px solid var(--success); }
    .verdict-box.bad { background: rgba(239, 68, 68, 0.1); border-left: 4px solid var(--danger); }

    /* API Endpoint Table */
    table { width: 100%; border-collapse: collapse; margin-top: 1rem; }
    th { text-align: left; padding: 0.75rem 1rem; color: var(--muted); font-size: 0.8rem; text-transform: uppercase; border-bottom: 1px solid #1e293b; }
    td { padding: 0.85rem 1rem; border-bottom: 1px solid #1a2233; font-size: 0.9rem; }
    .tag { padding: 0.25rem 0.5rem; border-radius: 6px; font-weight: 700; font-size: 0.75rem; }
    .tag-fast { background: rgba(16, 185, 129, 0.2); color: #34d399; }
    .tag-slow { background: rgba(245, 158, 11, 0.2); color: #fbbf24; }
    .tag-fail { background: rgba(239, 68, 68, 0.2); color: #f87171; }
  </style>
</head>
<body>
  <div class="container">
    <!-- Header -->
    <div class="header">
      <div class="badge">⚡ GO LOAD TEST STUDIO</div>
      <h1>Website Auto Stress Tester</h1>
      <p class="subtitle">১-ক্লিক করলেই স্বয়ংক্রিয়ভাবে ওয়েবসাইটের সব API টেস্ট করে সার্ভারের ব্রেকিং পয়েন্ট জানিয়ে দিবে।</p>
    </div>

    <!-- 1-Click Action Card -->
    <div class="action-card">
      <div class="input-group">
        <input type="text" id="targetBaseUrl" value="http://localhost:5000" placeholder="http://localhost:5000">
      </div>

      <button id="mainTestBtn" class="big-btn" onclick="toggleTest()">
        <span>🚀 START ALL API TEST</span>
      </button>

      <!-- Live progress during test -->
      <div id="progressBox" class="progress-box">
        <div class="progress-title">
          <span id="progressText">🔥 সব API-তে একসাথে লোড প্রেসার দেওয়া হচ্ছে...</span>
          <span id="elapsedTimer" style="color: var(--muted); font-family: monospace;">00:00</span>
        </div>
        <div class="live-stats-bar">
          <div class="mini-stat">
            <div class="mini-label">স্পিড (RPS)</div>
            <div id="liveRps" class="mini-val">0</div>
          </div>
          <div class="mini-stat">
            <div class="mini-label">একসাথে ভিজিটর (VUs)</div>
            <div id="liveWorkers" class="mini-val">0</div>
          </div>
          <div class="mini-stat">
            <div class="mini-label">গড় রেসপন্স টাইম</div>
            <div id="liveLat" class="mini-val">0ms</div>
          </div>
          <div class="mini-stat">
            <div class="mini-label">ফেইলিউর / এরর রেট</div>
            <div id="liveErr" class="mini-val">0%</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Result & Report Card (Human Readable) -->
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
    var isRunning = false;
    var eventSource = null;

    function initSSE() {
      if (eventSource) eventSource.close();
      eventSource = new EventSource('/api/events');

      eventSource.addEventListener('progress', function(e) {
        var snap = JSON.parse(e.data);
        updateProgress(snap);
      });

      eventSource.addEventListener('completed', function(e) {
        var res = JSON.parse(e.data);
        renderComplete(res);
      });

      eventSource.addEventListener('started', function() {
        setRunningState(true);
      });
    }

    function toggleTest() {
      if (isRunning) {
        fetch('/api/stop', { method: 'POST' });
        return;
      }

      var url = document.getElementById('targetBaseUrl').value || 'http://localhost:5000';
      document.getElementById('resultSection').style.display = 'none';
      setRunningState(true);

      fetch('/api/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ baseUrl: url })
      });
    }

    function setRunningState(running) {
      isRunning = running;
      var btn = document.getElementById('mainTestBtn');
      var pBox = document.getElementById('progressBox');

      if (running) {
        btn.classList.add('running');
        btn.innerHTML = '<span>🛑 STOP TEST</span>';
        pBox.style.display = 'block';
      } else {
        btn.classList.remove('running');
        btn.innerHTML = '<span>🚀 START ALL API TEST</span>';
      }
    }

    function updateProgress(snap) {
      document.getElementById('liveRps').textContent = Math.round(snap.CurrentRPS);
      document.getElementById('liveWorkers').textContent = snap.ActiveWorkers;
      document.getElementById('liveLat').textContent = formatLat(snap.CurrentAvgLat);
      document.getElementById('liveErr').textContent = snap.ErrorRate.toFixed(1) + '%';

      var sec = Math.floor(snap.Elapsed / 1000000000);
      var m = String(Math.floor(sec / 60)).padStart(2, '0');
      var s = String(sec % 60).padStart(2, '0');
      document.getElementById('elapsedTimer').textContent = m + ':' + s;
    }

    function formatLat(ns) {
      if (ns > 1000000000) return (ns / 1000000000).toFixed(1) + 's';
      if (ns > 1000000) return Math.round(ns / 1000000) + 'ms';
      return Math.round(ns / 1000) + 'µs';
    }

    function renderComplete(res) {
      setRunningState(false);
      var sec = document.getElementById('resultSection');
      sec.style.display = 'block';

      var verdict = document.getElementById('verdictText');
      var pill = document.getElementById('resultPill');

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

      // Endpoint table
      var tbody = document.getElementById('endpointTable');
      tbody.innerHTML = '';
      (res.endpoints || []).forEach(function(ep) {
        var tr = document.createElement('tr');
        var speedText = formatLat(ep.latency.median);
        var tagClass = 'tag tag-fast';
        var tagLabel = 'Fast (ভালো)';

        if (ep.errorRate > 5) {
          tagClass = 'tag tag-fail';
          tagLabel = 'Failed (' + ep.errorRate.toFixed(0) + '% Error)';
        } else if (ep.latency.median > 500000000) {
          tagClass = 'tag tag-slow';
          tagLabel = 'Slow (ধীরগতির)';
        }

        tr.innerHTML = '<td><strong>' + ep.name + '</strong><br><small style="color: #64748b;">' + ep.url + '</small></td>' +
          '<td>' + speedText + '</td>' +
          '<td><span class="' + tagClass + '">' + tagLabel + '</span></td>';
        tbody.appendChild(tr);
      });
    }

    initSSE();
  </script>
</body>
</html>`
}
