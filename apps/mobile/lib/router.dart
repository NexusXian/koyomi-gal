import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'providers/app_providers.dart';
import 'pages/articles/article_detail_page.dart';
import 'pages/articles/article_list_page.dart';
import 'pages/auth/login_page.dart';
import 'pages/auth/register_page.dart';
import 'pages/feedback/feedback_page.dart';
import 'pages/galgames/galgame_detail_page.dart';
import 'pages/galgames/galgame_form_page.dart';
import 'pages/galgames/galgame_list_page.dart';
import 'pages/home/home_page.dart';
import 'pages/home/shell_page.dart';
import 'pages/my/my_page.dart';
import 'pages/notifications/notifications_page.dart';
import 'pages/novels/novel_detail_page.dart';
import 'pages/novels/novel_form_page.dart';
import 'pages/novels/novel_list_page.dart';
import 'pages/novels/novel_volumes_page.dart';
import 'pages/novels/volume_detail_page.dart';
import 'pages/novels/volume_form_page.dart';
import 'pages/posts/post_detail_page.dart';
import 'pages/posts/post_form_page.dart';
import 'pages/posts/post_list_page.dart';
import 'pages/settings/settings_experience_page.dart';
import 'pages/settings/settings_privacy_page.dart';
import 'pages/settings/settings_profile_page.dart';
import 'pages/user/user_profile_page.dart';

final rootNavigatorKey = GlobalKey<NavigatorState>();

final _protectedPrefixes = [
  '/my',
  '/notifications',
  '/settings',
  '/galgames/new',
  '/posts/new',
  '/novels/new',
];

String? _authGuard(BuildContext context, GoRouterState state) {
  final auth = refContainer.read(authControllerProvider);
  if (auth.status == AuthStatus.loading) {
    return null;
  }
  final path = state.matchedLocation;
  final isProtected =
      _protectedPrefixes.any(path.startsWith) ||
      path.contains('/edit') ||
      path.contains('/volumes/new') ||
      path.contains('/volumes/') && path.endsWith('/edit');
  if (isProtected && auth.status != AuthStatus.authenticated) {
    return '/login?redirect=${Uri.encodeComponent(path)}';
  }
  return null;
}

/// Container reference for imperative auth reads inside guards.
late final ProviderContainer refContainer;

GoRouter buildRouter() {
  return GoRouter(
    navigatorKey: rootNavigatorKey,
    initialLocation: '/home',
    redirect: _authGuard,
    routes: [
      GoRoute(path: '/login', builder: (context, state) => const LoginPage()),
      GoRoute(
        path: '/register',
        builder: (context, state) => const RegisterPage(),
      ),
      StatefulShellRoute.indexedStack(
        builder: (context, state, navigationShell) {
          return ShellPage(navigationShell: navigationShell);
        },
        branches: [
          StatefulShellBranch(
            routes: [
            GoRoute(
              path: '/home',
              builder: (context, state) => const HomePage(),
            ),
            ],
          ),
          StatefulShellBranch(
            routes: [
            GoRoute(
              path: '/galgames',
              builder: (context, state) => const GalgameListPage(),
            ),
            ],
          ),
          StatefulShellBranch(
            routes: [
            GoRoute(
              path: '/posts',
              builder: (context, state) => const PostListPage(),
            ),
            ],
          ),
          StatefulShellBranch(
            routes: [
            GoRoute(
              path: '/novels',
              builder: (context, state) => const NovelListPage(),
            ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(path: '/my', builder: (context, state) => const MyPage()),
            ],
            ),
        ],
      ),
      GoRoute(
        path: '/galgames/new',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) => const GalgameFormPage(),
      ),
      GoRoute(
        path: '/galgames/:id',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) =>
            GalgameDetailPage(id: int.parse(state.pathParameters['id']!)),
      ),
      GoRoute(
        path: '/galgames/:id/edit',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) =>
            GalgameFormPage(editId: int.parse(state.pathParameters['id']!)),
      ),
      GoRoute(
        path: '/posts/new',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) => PostFormPage(
          galgameId: state.uri.queryParameters['galgameId'] != null
              ? int.tryParse(state.uri.queryParameters['galgameId']!)
              : null,
        ),
      ),
      GoRoute(
        path: '/posts/:id',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) =>
            PostDetailPage(id: int.parse(state.pathParameters['id']!)),
      ),
      GoRoute(
        path: '/posts/:id/edit',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) =>
            PostFormPage(editId: int.parse(state.pathParameters['id']!)),
      ),
      GoRoute(
        path: '/novels/new',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) => const NovelFormPage(),
      ),
      GoRoute(
        path: '/novels/:id',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) =>
            NovelDetailPage(id: int.parse(state.pathParameters['id']!)),
      ),
      GoRoute(
        path: '/novels/:id/edit',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) =>
            NovelFormPage(editId: int.parse(state.pathParameters['id']!)),
      ),
      GoRoute(
        path: '/novels/:id/volumes',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) =>
            NovelVolumesPage(novelId: int.parse(state.pathParameters['id']!)),
      ),
      GoRoute(
        path: '/novels/:id/volumes/new',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) =>
            VolumeFormPage(novelId: int.parse(state.pathParameters['id']!)),
      ),
      GoRoute(
        path: '/novels/:id/volumes/:volumeId',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) => VolumeDetailPage(
          novelId: int.parse(state.pathParameters['id']!),
          volumeId: int.parse(state.pathParameters['volumeId']!),
        ),
      ),
      GoRoute(
        path: '/novels/:id/volumes/:volumeId/edit',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) => VolumeFormPage(
          novelId: int.parse(state.pathParameters['id']!),
          volumeId: int.parse(state.pathParameters['volumeId']!),
        ),
      ),
      GoRoute(
        path: '/articles',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) =>
            ArticleListPage(type: state.uri.queryParameters['type']),
      ),
      GoRoute(
        path: '/articles/:id',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) =>
            ArticleDetailPage(id: int.parse(state.pathParameters['id']!)),
      ),
      GoRoute(
        path: '/user/:username',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) =>
            UserProfilePage(username: state.pathParameters['username']!),
      ),
      GoRoute(
        path: '/notifications',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) => const NotificationsPage(),
      ),
      GoRoute(
        path: '/settings/profile',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) => const SettingsProfilePage(),
      ),
      GoRoute(
        path: '/settings/privacy',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) => const SettingsPrivacyPage(),
      ),
      GoRoute(
        path: '/settings/experience',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) => const SettingsExperiencePage(),
      ),
      GoRoute(
        path: '/feedback',
        parentNavigatorKey: rootNavigatorKey,
        builder: (context, state) => const FeedbackPage(),
      ),
    ],
  );
}
