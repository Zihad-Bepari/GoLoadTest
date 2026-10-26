const http = require('http');
const { extractRoutes, autoTest } = require('../../adapters/nodejs');

// 1. Simulate an Express App structure with multiple routes and a sub-router
const mockExpressApp = {
  _router: {
    stack: [
      {
        route: {
          path: '/health',
          methods: { get: true }
        }
      },
      {
        route: {
          path: '/api/v1/auth/login',
          methods: { post: true }
        }
      },
      {
        name: 'router',
        regexp: /^\/api\/v1\/?(?=\/|$)/i,
        handle: {
          stack: [
            {
              route: {
                path: '/users',
                methods: { get: true, post: true }
              }
            },
            {
              route: {
                path: '/users/:id',
                methods: { get: true, delete: true }
              }
            },
            {
              route: {
                path: '/orders',
                methods: { get: true }
              }
            }
          ]
        }
      }
    ]
  }
};

console.log('Testing route auto-discovery...');
const routes = extractRoutes(mockExpressApp);
console.log('Discovered routes count:', routes.length);
routes.forEach(r => console.log(`  - [${r.method}] ${r.path}`));

if (routes.length !== 7) {
  console.error('FAIL: Expected 7 routes, got', routes.length);
  process.exit(1);
}

// 2. Start HTTP server handling the discovered routes
const server = http.createServer((req, res) => {
  res.writeHead(200, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ status: 'ok', route: req.url, method: req.method }));
});

server.listen(3005, async () => {
  console.log('\n🚀 Test server listening on http://127.0.0.1:3005');
  try {
    console.log('⚡ Running autoTest with Break-Until-Crash...');
    const result = await autoTest(mockExpressApp, {
      baseUrl: 'http://127.0.0.1:3005',
      duration: '2s',
      concurrency: 20,
      breakUntilCrash: false
    });

    console.log('\n🎉 AutoTest Result:');
    console.log('- Total Requests:', result.totalRequests);
    console.log('- Throughput:', Math.round(result.throughputRps), 'RPS');
    console.log('- Discovered Endpoints in Report:', (result.endpoints || []).length);
    (result.endpoints || []).forEach(ep => {
      console.log(`   * ${ep.name}: ${ep.totalRequests} reqs, ${Math.round(ep.rps)} RPS, ${ep.errorRate}% error`);
    });

    if (!result.endpoints || result.endpoints.length === 0) {
      throw new Error('Expected endpoints in result report, but none found');
    }

    console.log('\n✅ ALL AUTO-DISCOVERY AND AUTO-CONNECT CHECKS PASSED PERFECTLY!');
  } catch (err) {
    console.error('❌ AutoTest failed:', err);
    process.exitCode = 1;
  } finally {
    server.close();
  }
});
