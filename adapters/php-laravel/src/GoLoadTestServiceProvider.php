<?php

namespace GoLoadTest;

use Illuminate\Support\ServiceProvider;

/**
 * GoLoadTestServiceProvider — Laravel service provider for GoLoadTest.
 *
 * This provider is automatically discovered by Laravel when the package
 * is listed in composer.json's "extra.laravel.providers" array.
 *
 * It registers the GoLoadTest facade and the `test:load` Artisan command.
 *
 * Manual registration (if auto-discovery is disabled):
 * Add to config/app.php providers array:
 *   GoLoadTest\GoLoadTestServiceProvider::class,
 */
class GoLoadTestServiceProvider extends ServiceProvider
{
    /**
     * Register GoLoadTest bindings into the Laravel service container.
     */
    public function register(): void
    {
        $this->app->singleton(GoLoadTest::class, function () {
            return new GoLoadTest();
        });

        // Allow resolution by interface name for easier mocking in tests
        $this->app->alias(GoLoadTest::class, 'goloadtest');
    }

    /**
     * Bootstrap package services.
     * Registers the Artisan command when running in a console context.
     */
    public function boot(): void
    {
        if ($this->app->runningInConsole()) {
            $this->commands([
                Commands\LoadTestCommand::class,
            ]);
        }
    }

    /**
     * Get the services provided by the provider.
     *
     * @return array<string>
     */
    public function provides(): array
    {
        return [GoLoadTest::class, 'goloadtest'];
    }
}
