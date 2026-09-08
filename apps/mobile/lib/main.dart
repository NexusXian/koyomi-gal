import 'dart:async';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:path_provider/path_provider.dart';

import 'core/startup/startup_prompt_coordinator.dart';
import 'core/theme/app_theme.dart';
import 'providers/app_providers.dart';
import 'providers/message_providers.dart';
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
      builder: (context, child) => StartupPromptHost(
        navigatorKey: rootNavigatorKey,
        child: child ?? const SizedBox.shrink(),
      ),
    );
  }
}

class StartupPromptHost extends ConsumerStatefulWidget {
  const StartupPromptHost({
    super.key,
    required this.navigatorKey,
    required this.child,
  });

  final GlobalKey<NavigatorState> navigatorKey;
  final Widget child;

  @override
  ConsumerState<StartupPromptHost> createState() => _StartupPromptHostState();
}

class _StartupPromptHostState extends ConsumerState<StartupPromptHost>
    with WidgetsBindingObserver {
  Timer? _startupTimer;
  bool _startupDelayElapsed = false;
  bool _coldStartCheckPending = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) {
        return;
      }
      _startupTimer = Timer(const Duration(milliseconds: 750), () {
        _startupDelayElapsed = true;
        _check(StartupCheckTrigger.coldStart);
      });
    });
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _syncMessagesOnResume();
    }
    if (state == AppLifecycleState.resumed && _startupDelayElapsed) {
      _check(
        _coldStartCheckPending
            ? StartupCheckTrigger.coldStart
            : StartupCheckTrigger.resume,
      );
    }
  }

  /// App 回前台：必要时重连私信 WebSocket 并校正未读数。
  void _syncMessagesOnResume() {
    final auth = ref.read(authControllerProvider);
    if (!auth.isAuthenticated) {
      return;
    }
    ref.read(messageRealtimeProvider).resume();
    ref.read(unreadMessagesProvider).refresh();
  }

  void _check(StartupCheckTrigger trigger) {
    final lifecycleState = WidgetsBinding.instance.lifecycleState;
    if (!mounted ||
        lifecycleState != null && lifecycleState != AppLifecycleState.resumed) {
      return;
    }
    final context = widget.navigatorKey.currentContext;
    if (context == null) {
      if (trigger == StartupCheckTrigger.coldStart) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted && _coldStartCheckPending) {
            _check(StartupCheckTrigger.coldStart);
          }
        });
      }
      return;
    }
    if (trigger == StartupCheckTrigger.coldStart) {
      _coldStartCheckPending = false;
    }
    unawaited(
      ref
          .read(startupPromptCoordinatorProvider)
          .check(context, trigger: trigger),
    );
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _startupTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.child;
}

final routerProvider = Provider<GoRouter>((ref) => buildRouter());
