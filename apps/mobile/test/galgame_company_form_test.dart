import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/models/auth_models.dart';
import 'package:mobile/pages/galgames/galgame_form_page.dart';
import 'package:mobile/pages/galgames/galgame_detail_page.dart';
import 'package:mobile/pages/galgames/galgame_list_page.dart';
import 'package:mobile/providers/app_providers.dart';
import 'package:mobile/services/auth_service.dart';

void main() {
  testWidgets(
    'company-only manager reads a draft through admin and saves only relations',
    (tester) async {
      final api = _CompanyFormApi();
      await _pumpForm(
        tester,
        api,
        editing: true,
        permissions: ['company:manage'],
      );

      expect(api.reads, contains('/api/v1/admin/galgames/41'));
      expect(api.reads, isNot(contains('/api/v1/galgames/41')));
      expect(find.text('Draft game · ID 41'), findsOneWidget);
      expect(find.byType(TextFormField), findsNothing);
      expect(find.widgetWithText(TextButton, '保存'), findsNothing);

      await tester.tap(find.byType(DropdownButton<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.text('发行商').last);
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(OutlinedButton, '保存公司关联'));
      await tester.pumpAndSettle();

      expect(api.writes, hasLength(1));
      expect(api.writes.single.method, 'PUT');
      expect(api.writes.single.path, '/api/v1/admin/games/41/companies');
      expect(api.writes.single.data, {
        'companies': [
          {'company_id': 12, 'role': 'publisher'},
        ],
      });
      expect(find.text('公司关联已保存'), findsOneWidget);
    },
  );

  testWidgets(
    'creator retries failed bindings without another game create or update',
    (tester) async {
      final api = _CompanyFormApi()..failFirstBinding = true;
      await _pumpForm(
        tester,
        api,
        editing: false,
        permissions: ['galgame:create', 'company:manage'],
      );

      await _enterNewGame(tester);
      await tester.scrollUntilVisible(
        find.text('搜索并添加公司'),
        200,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.tap(find.text('搜索并添加公司'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Example Company').last);
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(TextButton, '保存'));
      await tester.pumpAndSettle();

      expect(find.text('游戏已保存，仅需重试公司关联。'), findsOneWidget);
      expect(find.byType(TextFormField), findsNothing);
      expect(api.bindingCalls, 1);

      await tester.tap(find.widgetWithText(TextButton, '保存公司关联'));
      await tester.pumpAndSettle();

      expect(find.byType(GalgameListPage), findsOneWidget);
      expect(find.text('已提交，等待审核'), findsOneWidget);
      expect(api.reads, contains('/api/v1/galgames'));
      expect(api.reads, isNot(contains('/api/v1/galgames/77')));
      expect(
        api.writes.where((request) => request.path == '/api/v1/galgames'),
        hasLength(1),
      );
      expect(
        api.writes.where((request) => request.path == '/api/v1/galgames/77'),
        isEmpty,
      );
      final bindings = api.writes
          .where(
            (request) => request.path == '/api/v1/admin/games/77/companies',
          )
          .toList();
      expect(bindings, hasLength(2));
      expect(bindings.first.data, bindings.last.data);
      expect(tester.takeException(), isNull);
    },
  );

  for (final status in [0, 1]) {
    testWidgets('creation status $status chooses the appropriate destination', (
      tester,
    ) async {
      final api = _CompanyFormApi()..createdStatus = status;
      await _pumpForm(
        tester,
        api,
        editing: false,
        permissions: ['galgame:create'],
      );
      await _enterNewGame(tester);
      await tester.tap(find.widgetWithText(TextButton, '保存'));
      await tester.pumpAndSettle();

      if (status == 1) {
        expect(find.byType(GalgameDetailPage), findsOneWidget);
        expect(api.reads, contains('/api/v1/galgames/77'));
      } else {
        expect(find.byType(GalgameListPage), findsOneWidget);
        expect(find.text('已提交，等待审核'), findsOneWidget);
        expect(api.reads, isNot(contains('/api/v1/galgames/77')));
      }
      expect(api.writes, hasLength(1));
      expect(api.writes.single.method, 'POST');
      expect(tester.takeException(), isNull);
    });
  }
}

Future<void> _enterNewGame(WidgetTester tester) async {
  await tester.enterText(
    find.byWidgetPredicate(
      (widget) => widget is TextField && widget.decoration?.labelText == '标题 *',
    ),
    'New game',
  );
  await tester.enterText(
    find.byWidgetPredicate(
      (widget) =>
          widget is TextField &&
          widget.decoration?.labelText == 'Slug *（字母、数字、-、_）',
    ),
    'new-game',
  );
}

Future<void> _pumpForm(
  WidgetTester tester,
  _CompanyFormApi api, {
  required bool editing,
  required List<String> permissions,
}) async {
  tester.view.physicalSize = const Size(800, 1200);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  final router = GoRouter(
    initialLocation: editing ? '/galgames/41/edit' : '/galgames/new',
    routes: [
      GoRoute(
        path: '/galgames/new',
        builder: (_, _) => const GalgameFormPage(),
      ),
      GoRoute(
        path: '/galgames/:id/edit',
        builder: (_, state) =>
            GalgameFormPage(editId: int.parse(state.pathParameters['id']!)),
      ),
      GoRoute(path: '/galgames', builder: (_, _) => const GalgameListPage()),
      GoRoute(
        path: '/galgames/:id',
        builder: (_, state) =>
            GalgameDetailPage(id: int.parse(state.pathParameters['id']!)),
      ),
    ],
  );
  addTearDown(router.dispose);
  addTearDown(() => api.dio.close(force: true));
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        apiClientProvider.overrideWithValue(api),
        authControllerProvider.overrideWith(
          (_) =>
              AuthController(AuthService(api))
                ..status = AuthStatus.unauthenticated,
        ),
        mePermissionsProvider.overrideWith(
          (_) async => MePermissions(permissions: permissions),
        ),
      ],
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pumpAndSettle();
}

class _CompanyFormApi extends ApiClient {
  _CompanyFormApi()
    : super(
        baseUrl: 'https://example.test',
        getAccessToken: () => null,
        refreshSession: () async => null,
        onSessionInvalid: () {},
      );

  final reads = <String>[];
  final writes = <({String method, String path, Object? data})>[];
  bool failFirstBinding = false;
  int bindingCalls = 0;
  int createdStatus = 0;

  static const _company = {
    'id': 12,
    'name': 'Example Company',
    'slug': 'example-company',
    'role': 'developer',
  };

  @override
  Future<dynamic> get(
    String path, {
    Map<String, dynamic>? queryParameters,
    ApiRequestFlags? flags,
  }) async {
    reads.add(path);
    switch (path) {
      case '/api/v1/tags':
      case '/api/v1/developers':
        return <Object>[];
      case '/api/v1/admin/galgames/41':
        return {
          'id': 41,
          'title': 'Draft game',
          'slug': 'draft-game',
          'status': 0,
          'companies': [_company],
        };
      case '/api/v1/companies':
        return {
          'items': [_company],
          'total': 1,
          'page': 1,
          'page_size': 20,
        };
      case '/api/v1/galgames':
        return {'items': <Object>[], 'total': 0, 'page': 1, 'limit': 20};
      case '/api/v1/galgames/77':
        if (createdStatus != 1) throw StateError('Game is not published');
        return {
          'id': 77,
          'title': 'New game',
          'slug': 'new-game',
          'status': createdStatus,
        };
      case '/api/v1/galgames/77/characters':
        return {'items': <Object>[]};
      default:
        throw StateError('Unexpected GET $path');
    }
  }

  @override
  Future<dynamic> post(
    String path, {
    Object? data,
    ApiRequestFlags? flags,
  }) async {
    writes.add((method: 'POST', path: path, data: data));
    if (path != '/api/v1/galgames') throw StateError('Unexpected POST $path');
    return {
      'id': 77,
      'title': 'New game',
      'slug': 'new-game',
      'status': createdStatus,
    };
  }

  @override
  Future<dynamic> put(
    String path, {
    Object? data,
    ApiRequestFlags? flags,
  }) async {
    writes.add((method: 'PUT', path: path, data: data));
    if (path == '/api/v1/admin/games/41/companies' ||
        path == '/api/v1/admin/games/77/companies') {
      bindingCalls++;
      if (failFirstBinding && bindingCalls == 1) throw StateError('Offline');
      return null;
    }
    throw StateError('Unauthorized metadata update $path');
  }
}
