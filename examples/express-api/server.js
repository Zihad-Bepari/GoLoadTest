const http = require('http');
const { runLoadTest } = require('../../adapters/nodejs');

// Simple native HTTP server simulating Express.js
let userCount = 0;
const server = http.createServer((req, res) => {
  if (req.url === '/api/users' && req.method === 'GET') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify([{ id: 1, name: 'Node User' }]));
  } else if (req.url === '/api/users' && req.method === 'POST') {
    userCount++;
    res.writeHead(201, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ id: userCount, status: 'created' }));
  } else {
    res.writeHead(404);
    res.end();
  }
});

server.listen(3000, async () => {
  console.log('🚀 Express mock server listening on http://localhost:3000');

  try {
    console.log('⚡ Running GoLoadTest via Node.js adapter...');
    const result = await runLoadTest({
      name: 'Express Users API Load Test',
      url: 'http://localhost:3000/api/users',
      duration: '3s',
      concurrency: 15,
      thresholds: {
        maxP95: '50ms',
        maxErrorRate: 1.0
      }
    });

    console.log('\n🎉 Node.js Load Test Result:');
    console.log(`- Total Requests: ${result.totalRequests}`);
    console.log(`- Throughput: ${result.throughputRps} RPS`);
    console.log(`- P95 Latency: ${result.latency.p95} ns`);
    console.log(`- Passed: ${result.passed}`);
  } catch (err) {
    console.error('Test failed:', err);
  } finally {
    server.close();
  }
});
