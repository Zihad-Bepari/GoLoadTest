package server

func getDashboardHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>GoLoadTest · Autonomous API Stress & Capacity Tester</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&family=JetBrains+Mono:wght@500;700&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #f8fafc;
      --card: #ffffff;
      --border: #e2e8f0;
      --primary: #2563eb;
      --primary-hover: #1d4ed8;
      --success: #16a34a;
      --danger: #dc2626;
      --warning: #d97706;
      --text: #0f172a;
      --muted: #64748b;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: var(--bg);
      color: var(--text);
      font-family: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      padding: 2.5rem 1rem;
      display: flex;
      justify-content: center;
      min-height: 100vh;
    }
    .container { max-width: 980px; width: 100%; }
    
    /* Top Header */
    .header { text-align: center; margin-bottom: 2rem; }
    .badge {
      display: inline-block;
      background: #eff6ff;
      color: #2563eb;
      padding: 0.35rem 0.95rem;
      border-radius: 9999px;
      font-size: 0.82rem;
      font-weight: 700;
      letter-spacing: 0.5px;
      margin-bottom: 0.75rem;
      border: 1px solid #bfdbfe;
    }
    h1 { font-size: 2.2rem; font-weight: 800; color: #0f172a; margin-bottom: 0.5rem; letter-spacing: -0.5px; }
    p.subtitle { color: #64748b; font-size: 1.05rem; max-width: 680px; margin: 0 auto; line-height: 1.6; }

    /* Main Big Action Card */
    .action-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 20px;
      padding: 2.5rem 2rem;
      text-align: center;
      box-shadow: 0 10px 30px -5px rgba(0,0,0,0.05), 0 4px 12px -2px rgba(0,0,0,0.03);
      margin-bottom: 2rem;
      position: relative;
    }

    .input-group {
      max-width: 520px;
      margin: 0 auto 1.5rem auto;
      display: flex;
      gap: 0.5rem;
      background: #f8fafc;
      padding: 0.4rem;
      border-radius: 12px;
      border: 1px solid #cbd5e1;
    }
    .input-group input {
      flex: 1;
      background: transparent;
      border: none;
      outline: none;
      color: #0f172a;
      padding: 0.6rem 1rem;
      font-size: 1rem;
      font-family: inherit;
    }

    /* Giant 1-Click Button */
    .big-btn {
      background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
      color: #ffffff;
      font-size: 1.25rem;
      font-weight: 800;
      padding: 1.15rem 2.8rem;
      border-radius: 14px;
      border: none;
      cursor: pointer;
      box-shadow: 0 8px 24px rgba(37, 99, 235, 0.35);
      transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
      display: inline-flex;
      align-items: center;
      gap: 0.75rem;
    }
    .big-btn:hover { transform: translateY(-2px); box-shadow: 0 12px 28px rgba(37, 99, 235, 0.45); }
    .big-btn:active { transform: translateY(0); }
    .big-btn.running {
      background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
      box-shadow: 0 8px 24px rgba(220, 38, 38, 0.35);
    }

    /* Live Progress Bar */
    .progress-box {
      display: none;
      margin-top: 1.75rem;
      background: #f8fafc;
      border: 1px solid #bfdbfe;
      border-radius: 14px;
      padding: 1.5rem;
      text-align: left;
    }
    .progress-title {
      font-weight: 700;
      color: #1e40af;
      display: flex;
      justify-content: space-between;
      margin-bottom: 1rem;
      font-size: 0.95rem;
    }
    .live-stats-bar {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 0.75rem;
    }
    .mini-stat {
      background: #ffffff;
      padding: 1rem 0.75rem;
      border-radius: 10px;
      border: 1px solid #e2e8f0;
      text-align: center;
    }
    .mini-label { font-size: 0.75rem; font-weight: 700; color: #64748b; text-transform: uppercase; letter-spacing: 0.5px; }
    .mini-val { font-size: 1.35rem; font-weight: 800; color: #0f172a; margin-top: 0.3rem; }

    /* Live System Defense Console */
    .defense-card {
      background: #ffffff;
      border: 1px solid #cbd5e1;
      border-radius: 14px;
      padding: 1.25rem;
      margin-top: 1.25rem;
      text-align: left;
      box-shadow: 0 4px 12px rgba(0,0,0,0.03);
    }
    .defense-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 1rem;
      flex-wrap: wrap;
      gap: 0.5rem;
    }
    .source-pill {
      font-size: 0.72rem;
      padding: 0.2rem 0.6rem;
      border-radius: 9999px;
      background: #eff6ff;
      color: #1d4ed8;
      font-weight: 700;
      border: 1px solid #bfdbfe;
    }
    .status-badge {
      font-size: 0.75rem;
      font-weight: 800;
      padding: 0.25rem 0.75rem;
      border-radius: 9999px;
      letter-spacing: 0.4px;
    }
    .badge-optimal { background: #dcfce7; color: #15803d; border: 1px solid #bbf7d0; }
    .badge-guarded { background: #fef9c3; color: #854d0e; border: 1px solid #fef08a; }
    .badge-throttled { background: #ffedd5; color: #c2410c; border: 1px solid #fed7aa; }
    .badge-overload { background: #fee2e2; color: #b91c1c; border: 1px solid #fca5a5; }

    .defense-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
      gap: 0.85rem;
    }
    .defense-subcard {
      background: #f8fafc;
      border: 1px solid #e2e8f0;
      border-radius: 12px;
      padding: 1rem;
    }
    .subcard-title {
      font-size: 0.85rem;
      font-weight: 700;
      color: #1e293b;
      margin-bottom: 0.75rem;
      display: flex;
      align-items: center;
      justify-content: space-between;
    }
    .meter-row {
      display: flex;
      justify-content: space-between;
      font-size: 0.8rem;
      margin-bottom: 0.35rem;
    }
    .meter-label { color: #64748b; font-weight: 600; }
    .meter-val { font-weight: 700; color: #0f172a; }
    .meter-bar-track {
      width: 100%;
      height: 7px;
      background: #e2e8f0;
      border-radius: 999px;
      overflow: hidden;
      margin-bottom: 0.75rem;
    }
    .meter-bar-fill {
      height: 100%;
      background: linear-gradient(90deg, #3b82f6, #1d4ed8);
      border-radius: 999px;
      transition: width 0.3s ease;
    }
    .guard-fill {
      background: linear-gradient(90deg, #10b981, #059669);
    }
    .stat-dual {
      display: flex;
      justify-content: space-between;
      font-size: 0.8rem;
      background: #ffffff;
      padding: 0.5rem 0.65rem;
      border-radius: 8px;
      border: 1px solid #edf2f7;
    }
    .stat-dual small {
      color: #64748b;
      display: block;
      margin-bottom: 0.15rem;
      font-size: 0.72rem;
      font-weight: 600;
      text-transform: uppercase;
    }

    /* Results Card */
    .result-section { display: none; }
    .summary-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 16px;
      padding: 2rem;
      margin-bottom: 1.5rem;
      box-shadow: 0 4px 20px -2px rgba(0, 0, 0, 0.04);
    }
    .summary-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 1.5rem;
    }
    .status-pill {
      padding: 0.45rem 1rem;
      border-radius: 9999px;
      font-size: 0.85rem;
      font-weight: 700;
      letter-spacing: 0.5px;
    }
    .status-ok { background: #dcfce7; color: #15803d; border: 1px solid #bbf7d0; }
    .status-crash { background: #fee2e2; color: #b91c1c; border: 1px solid #fecaca; }

    .verdict-box {
      border-radius: 12px;
      padding: 1.25rem 1.5rem;
      font-size: 1rem;
      line-height: 1.6;
      margin-bottom: 1.75rem;
    }
    .verdict-box.good { background: #f0fdf4; border: 1px solid #bbf7d0; color: #166534; }
    .verdict-box.bad { background: #fef2f2; border: 1px solid #fecaca; color: #991b1b; }

    /* Defense Summary in Results */
    .defense-summary-box {
      background: #f8fafc;
      border: 1px solid #cbd5e1;
      border-radius: 12px;
      padding: 1.25rem;
      margin-bottom: 1.75rem;
    }
    .defense-summary-title {
      font-size: 1rem;
      font-weight: 700;
      color: #0f172a;
      margin-bottom: 0.75rem;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }
    .defense-kpi-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 0.75rem;
    }
    .defense-kpi {
      background: #ffffff;
      padding: 0.85rem;
      border-radius: 10px;
      border: 1px solid #e2e8f0;
      text-align: center;
    }
    .defense-kpi-label { font-size: 0.72rem; color: #64748b; font-weight: 700; text-transform: uppercase; }
    .defense-kpi-val { font-size: 1.25rem; font-weight: 800; color: #1e293b; margin-top: 0.25rem; }

    /* Endpoint Table */
    table { width: 100%; border-collapse: collapse; text-align: left; font-size: 0.95rem; }
    th { padding: 0.85rem 1rem; background: #f8fafc; color: #475569; font-weight: 700; font-size: 0.85rem; border-bottom: 1px solid var(--border); }
    td { padding: 1rem; border-bottom: 1px solid #f1f5f9; color: #1e293b; }
    tr:last-child td { border-bottom: none; }
    .tag { display: inline-block; padding: 0.25rem 0.65rem; border-radius: 6px; font-size: 0.75rem; font-weight: 700; }
    .tag-fast { background: #dcfce7; color: #15803d; }
    .tag-slow { background: #fef9c3; color: #854d0e; }
    .tag-fail { background: #fee2e2; color: #b91c1c; }

    @media (max-width: 640px) {
      .live-stats-bar { grid-template-columns: repeat(2, 1fr); }
      .defense-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="container">
    <!-- Header -->
    <div class="header">
      <span class="badge">RESILIENT API TESTING &amp; DEFENSE MONITOR</span>
      <h1>GoLoadTest Live Command Center</h1>
      <p class="subtitle">Autonomous breaking-point discovery engine with live in-process GoLoadGuard &amp; GoCacheShield telemetry.</p>
    </div>

    <!-- 1-Click Action Card -->
    <div class="action-card">
      <div class="input-group">
        <input type="text" id="targetBaseUrl" value="http://localhost:5000" placeholder="http://localhost:5000">
      </div>

      <button id="mainTestBtn" class="big-btn" onclick="toggleTest()">
        <span>🚀 START ALL API STRESS TEST</span>
      </button>

      <!-- Live progress during test -->
      <div id="progressBox" class="progress-box">
        <div class="progress-title">
          <span id="progressText">🔥 Ramping load pressure across all routes...</span>
          <span id="elapsedTimer" style="color: var(--muted); font-family: monospace;">00:00</span>
        </div>
        
        <div class="live-stats-bar">
          <div class="mini-stat">
            <div class="mini-label">Speed (RPS)</div>
            <div id="liveRps" class="mini-val">0</div>
          </div>
          <div class="mini-stat">
            <div class="mini-label">Active Users (VUs)</div>
            <div id="liveWorkers" class="mini-val">0</div>
          </div>
          <div class="mini-stat">
            <div class="mini-label">Avg Latency</div>
            <div id="liveLat" class="mini-val">0ms</div>
          </div>
          <div class="mini-stat">
            <div class="mini-label">Error Rate</div>
            <div id="liveErr" class="mini-val">0%</div>
          </div>
        </div>

        <!-- Live Server Vitals & Defense Monitoring -->
        <div class="defense-card">
          <div class="defense-header">
            <div style="display:flex; align-items:center; gap:0.5rem;">
              <span style="font-size:1.1rem;">🛡️</span>
              <strong>Live Server Vitals &amp; Resilience Telemetry</strong>
            </div>
            <div style="display:flex; align-items:center; gap:0.5rem;">
              <span id="telemetrySourcePill" class="source-pill">Detecting telemetry...</span>
              <span id="telemetryStatusBadge" class="status-badge badge-optimal">🟢 OPTIMAL</span>
            </div>
          </div>

          <div class="defense-grid">
            <!-- 1. Server Host Vitals -->
            <div class="defense-subcard">
              <div class="subcard-title">
                <span>🖥️ Server Vitals</span>
                <small style="color:#64748b; font-weight:normal;" id="liveVitalsSource">OS Metrics</small>
              </div>
              <div class="meter-row">
                <span class="meter-label">CPU Usage</span>
                <span id="liveCpuText" class="meter-val">0%</span>
              </div>
              <div class="meter-bar-track">
                <div id="liveCpuBar" class="meter-bar-fill" style="width: 0%;"></div>
              </div>
              <div class="stat-dual">
                <div>
                  <small>Memory (RAM)</small>
                  <div id="liveRamText" style="font-weight:700;">--</div>
                </div>
                <div>
                  <small>Goroutines/Tasks</small>
                  <div id="liveRoutinesText" style="font-weight:700;">--</div>
                </div>
              </div>
            </div>

            <!-- 2. GoLoadGuard Concurrency Defense -->
            <div class="defense-subcard">
              <div class="subcard-title">
                <span>🛡️ GoLoadGuard</span>
                <span id="guardActivePill" class="tag tag-fast" style="font-size:0.68rem;">Active</span>
              </div>
              <div class="meter-row">
                <span class="meter-label">In-Flight Concurrency</span>
                <span id="liveInFlightText" class="meter-val">0 active</span>
              </div>
              <div class="meter-bar-track">
                <div id="liveInFlightBar" class="meter-bar-fill guard-fill" style="width: 0%;"></div>
              </div>
              <div class="stat-dual">
                <div>
                  <small>FIFO Queue</small>
                  <div id="liveQueueText" style="font-weight:700;">0 waiting</div>
                </div>
                <div>
                  <small>503 Shed Drops</small>
                  <div id="liveRejectedText" style="font-weight:700; color:#16a34a;">0 (Crash Prevented)</div>
                </div>
              </div>
            </div>

            <!-- 3. GoCacheShield Data Defense -->
            <div class="defense-subcard">
              <div class="subcard-title">
                <span>⚡ GoCacheShield</span>
                <span id="shieldActivePill" class="tag tag-fast" style="font-size:0.68rem;">Active</span>
              </div>
              <div class="meter-row">
                <span class="meter-label">Stampede Merged</span>
                <span id="liveCoalescedText" class="meter-val">0 reqs</span>
              </div>
              <div class="stat-dual" style="margin-bottom:0.4rem;">
                <div>
                  <small>DB Queries Saved</small>
                  <div id="liveSavedDbText" style="font-weight:700; color:#2563eb;">0 queries</div>
                </div>
                <div>
                  <small>Cache Hits</small>
                  <div id="liveCacheHitsText" style="font-weight:700;">0 hits</div>
                </div>
              </div>
              <div class="stat-dual">
                <div>
                  <small>Penetration Blocked</small>
                  <div id="liveNegHitsText" style="font-weight:700;">0 attacks</div>
                </div>
                <div>
                  <small>DB Bulkhead Pool</small>
                  <div id="liveBulkheadText" style="font-weight:700;">0 active</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Result & Report Card -->
    <div id="resultSection" class="result-section">
      <div class="summary-card">
        <div class="summary-header">
          <h2 style="font-size: 1.3rem; color: #0f172a;">📊 Executive Test Results</h2>
          <span id="resultPill" class="status-pill status-ok">COMPLETED</span>
        </div>

        <div id="verdictText" class="verdict-box good">
          Stress test completed successfully.
        </div>

        <!-- Defense Summary Scorecard -->
        <div id="defenseSummaryCard" class="defense-summary-box">
          <div class="defense-summary-title">
            <span>🛡️ System Defense &amp; Resilience Impact</span>
            <span id="defenseScoreBadge" class="tag tag-fast">100% PROTECTED</span>
          </div>
          <div class="defense-kpi-grid">
            <div class="defense-kpi">
              <div class="defense-kpi-label">DB Queries Spared</div>
              <div id="kpiSavedDb" class="defense-kpi-val" style="color: #2563eb;">0</div>
            </div>
            <div class="defense-kpi">
              <div class="defense-kpi-label">Crash Drops Handled (503)</div>
              <div id="kpiShedDrops" class="defense-kpi-val" style="color: #16a34a;">0</div>
            </div>
            <div class="defense-kpi">
              <div class="defense-kpi-label">Stampede Merges</div>
              <div id="kpiStampedes" class="defense-kpi-val">0</div>
            </div>
            <div class="defense-kpi">
              <div class="defense-kpi-label">Penetration Attacks Blocked</div>
              <div id="kpiPenetrations" class="defense-kpi-val">0</div>
            </div>
          </div>
        </div>

        <h3 style="font-size: 1.1rem; color: #0f172a; margin-top: 1.75rem; margin-bottom: 0.75rem;">🔍 Endpoint Performance Breakdown</h3>
        <div style="overflow-x: auto;">
          <table>
            <thead>
              <tr>
                <th>Endpoint &amp; Path</th>
                <th>Response Latency</th>
                <th>Status</th>
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
        btn.innerHTML = '<span>🚀 START ALL API STRESS TEST</span>';
      }
    }

    function updateProgress(snap) {
      var rps = snap.CurrentRPS !== undefined ? snap.CurrentRPS : (snap.currentRps || 0);
      var workers = snap.ActiveWorkers !== undefined ? snap.ActiveWorkers : (snap.activeWorkers || 0);
      var lat = snap.CurrentAvgLat !== undefined ? snap.CurrentAvgLat : (snap.currentAvgLat || 0);
      var err = snap.ErrorRate !== undefined ? snap.ErrorRate : (snap.errorRate || 0);
      var elapsed = snap.Elapsed !== undefined ? snap.Elapsed : (snap.elapsed || 0);

      document.getElementById('liveRps').textContent = Math.round(rps);
      document.getElementById('liveWorkers').textContent = workers;
      document.getElementById('liveLat').textContent = formatLat(lat);
      document.getElementById('liveErr').textContent = err.toFixed(1) + '%';

      var sec = Math.floor(elapsed / 1000000000);
      var m = String(Math.floor(sec / 60)).padStart(2, '0');
      var s = String(sec % 60).padStart(2, '0');
      document.getElementById('elapsedTimer').textContent = m + ':' + s;

      // Update Telemetry
      var telem = snap.Telemetry || snap.telemetry;
      if (telem) {
        updateTelemetryUI(telem, err);
      }
    }

    function updateTelemetryUI(t, clientErrRate) {
      var sourcePill = document.getElementById('telemetrySourcePill');
      var statusBadge = document.getElementById('telemetryStatusBadge');
      if (t.source) {
        sourcePill.textContent = t.source;
      }

      var status = t.serverStatus || 'OPTIMAL';
      if (clientErrRate > 50) status = 'OVERLOADED';
      else if (t.rejectedRequests > 0) status = 'THROTTLED';
      else if (t.queueLength > 0 || (t.inFlightRequests > 0 && t.maxConcurrency > 0 && t.inFlightRequests >= t.maxConcurrency)) status = 'GUARDED';

      statusBadge.textContent = '🟢 ' + status;
      statusBadge.className = 'status-badge badge-optimal';
      if (status === 'GUARDED') {
        statusBadge.textContent = '🟡 ' + status;
        statusBadge.className = 'status-badge badge-guarded';
      } else if (status === 'THROTTLED') {
        statusBadge.textContent = '🟠 ' + status + ' (SHEDDING)';
        statusBadge.className = 'status-badge badge-throttled';
      } else if (status === 'OVERLOADED') {
        statusBadge.textContent = '🔴 ' + status;
        statusBadge.className = 'status-badge badge-overload';
      }

      // Server Vitals
      var cpu = t.cpuPercent || 0;
      document.getElementById('liveCpuText').textContent = cpu.toFixed(1) + '%';
      document.getElementById('liveCpuBar').style.width = Math.min(100, Math.max(0, cpu)) + '%';
      document.getElementById('liveRamText').textContent = t.memoryMb > 0 ? (t.memoryMb.toFixed(0) + ' MB') : 'Healthy';
      document.getElementById('liveRoutinesText').textContent = t.activeGoroutines > 0 ? t.activeGoroutines : (workersCount || '--');

      // GoLoadGuard
      var inFlight = t.inFlightRequests || 0;
      var maxCon = t.maxConcurrency || 50;
      var inFlightPct = maxCon > 0 ? (inFlight / maxCon) * 100 : 0;
      document.getElementById('liveInFlightText').textContent = inFlight + (maxCon > 0 ? (' / ' + maxCon) : ' active');
      document.getElementById('liveInFlightBar').style.width = Math.min(100, inFlightPct) + '%';
      document.getElementById('liveQueueText').textContent = (t.queueLength || 0) + ' waiting';
      document.getElementById('liveRejectedText').textContent = (t.rejectedRequests || 0) + ' saved';

      // GoCacheShield
      var coalesced = t.coalescedRequests || 0;
      var savedDb = t.savedDbQueries || (coalesced > 0 ? coalesced - 1 : 0);
      var cacheHits = t.cacheHits || 0;
      var negHits = t.negativeCacheHits || 0;
      var bulkhead = t.activeDbBulkhead || 0;

      document.getElementById('liveCoalescedText').textContent = coalesced + ' merged';
      document.getElementById('liveSavedDbText').textContent = savedDb + ' saved';
      document.getElementById('liveCacheHitsText').textContent = cacheHits + ' hits';
      document.getElementById('liveNegHitsText').textContent = negHits + ' attacks';
      document.getElementById('liveBulkheadText').textContent = bulkhead + ' active';
    }

    var workersCount = 0;

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
        pill.textContent = 'BREAKING POINT IDENTIFIED';
        verdict.className = 'verdict-box bad';
        verdict.innerHTML = '⚠️ <strong>Server Breaking Point Identified:</strong> Your server safely sustained up to <strong>' +
          res.breakingPoint.maxStableConcurrency + ' concurrent users</strong>. Beyond this load, at <strong>' +
          res.breakingPoint.crashConcurrency + ' concurrent users</strong>, the server began dropping requests or failing.<br>Reason: ' + res.breakingPoint.crashReason;
      } else {
        pill.className = 'status-pill status-ok';
        pill.textContent = 'ALL PASSED & RESILIENT';
        verdict.className = 'verdict-box good';
        verdict.innerHTML = '🎉 <strong>Excellent!</strong> Your server successfully sustained maximum traffic across all tested routes without crashes or excessive slowdown.';
      }

      // Populate Defense Scorecard in Results
      var telem = res.defenseTelemetry || res.DefenseTelemetry;
      if (telem) {
        var saved = telem.savedDbQueries || (telem.coalescedRequests > 0 ? telem.coalescedRequests - 1 : 0);
        document.getElementById('kpiSavedDb').textContent = saved > 0 ? saved.toLocaleString() : '0';
        document.getElementById('kpiShedDrops').textContent = (telem.rejectedRequests || 0).toLocaleString();
        document.getElementById('kpiStampedes').textContent = (telem.coalescedRequests || 0).toLocaleString();
        document.getElementById('kpiPenetrations').textContent = (telem.negativeCacheHits || 0).toLocaleString();
      }

      // Endpoint table
      var tbody = document.getElementById('endpointTable');
      tbody.innerHTML = '';
      (res.endpoints || []).forEach(function(ep) {
        var tr = document.createElement('tr');
        var speedText = formatLat(ep.latency.median);
        var tagClass = 'tag tag-fast';
        var tagLabel = 'Fast';

        if (ep.errorRate > 5) {
          tagClass = 'tag tag-fail';
          tagLabel = 'Failed (' + ep.errorRate.toFixed(0) + '% Error)';
        } else if (ep.latency.median > 500000000) {
          tagClass = 'tag tag-slow';
          tagLabel = 'Slow';
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
