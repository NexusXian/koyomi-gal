import 'package:cookie_jar/cookie_jar.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:path_provider/path_provider.dart';

import 'core/theme/app_theme.dart';
import 'providers/app_providers.dart';
import 'router.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // path_provider has no web implementation; on web the browser itself
  // manages the httpOnly refresh cookie (dio sends withCredentials).
  final CookieJar? cookieJar;
  if (!kIsWeb) {
    final supportDir = await getApplicationSupportDirectory();
    cookieJar = PersistCookieJar(
      storage: FileStorage('${supportDir.path}/cookies'),
    );
  } else {
    cookieJar = null;
  }

  final container = ProviderContainer(
    overrides: [cookieJarProvider.overrideWithValue(cookieJar)],
  );
  refContainer = container;

  final themeController = container.read(themeControllerProvider);
  await loadThemePreference(themeController);

  // Re-evaluate the auth guard when the session restores / changes.
  container.listen<AuthController>(
    authControllerProvider,
    (_, _) => container.read(routerProvider).refresh(),
    fireImmediately: false,
  );

  runApp(
    UncontrolledProviderScope(
      container: container,
      child: const KoyomiApp(),
    ),
  );
}

class KoyomiApp extends ConsumerWidget {
  const KoyomiApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    final themeMode = ref.watch(
      themeControllerProvider.select((controller) => controller.mode),
    );

    return MaterialApp.router(
      title: 'Koyomi Gal',
      debugShowCheckedModeBanner: false,
      theme: buildLightTheme(),
      darkTheme: buildDarkTheme(),
      themeMode: themeMode,
      routerConfig: router,
    );
  }
}

final routerProvider = Provider<GoRouter>((ref) => buildRouter());
