/// Runtime configuration, overridable via --dart-define at build time.
class AppConfig {
  AppConfig._();

  static const apiBase = String.fromEnvironment(
    'API_BASE',
    defaultValue: 'https://api.koyomigal.xyz',
  );

  static const apiPrefix = '/api/v1';
}
