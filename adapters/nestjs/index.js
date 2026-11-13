'use strict';

const { runLoadTest, autoTest, extractRoutes } = require('../nodejs');

let common;
try {
  common = require('@nestjs/common');
} catch (_) {}

class GoLoadTestService {
  async run(options) {
    return runLoadTest(options);
  }

  async autoTest(app, options = {}) {
    let baseUrl = options.baseUrl;
    if (!baseUrl && typeof app.getUrl === 'function') {
      try {
        baseUrl = await app.getUrl();
        baseUrl = baseUrl.replace('[::1]', '127.0.0.1').replace('0.0.0.0', '127.0.0.1');
      } catch (_) {}
    }

    const instance = (app && typeof app.getHttpAdapter === 'function')
      ? app.getHttpAdapter().getInstance()
      : (app && typeof app.getHttpServer === 'function' ? app.getHttpServer() : app);

    return autoTest(instance, { ...options, baseUrl });
  }

  extractRoutes(app) {
    const instance = (app && typeof app.getHttpAdapter === 'function')
      ? app.getHttpAdapter().getInstance()
      : (app && typeof app.getHttpServer === 'function' ? app.getHttpServer() : app);
    return extractRoutes(instance);
  }
}

if (common && typeof common.Injectable === 'function') {
  common.Injectable()(GoLoadTestService);
}

class GoLoadTestModule {}
if (common && typeof common.Module === 'function') {
  common.Module({
    providers: [GoLoadTestService],
    exports: [GoLoadTestService],
  })(GoLoadTestModule);
}

module.exports = {
  GoLoadTestService,
  GoLoadTestModule,
};
