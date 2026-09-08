import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/core/startup/startup_prompt_coordinator.dart';
import 'package:mobile/features/announcement/models/announcement.dart';
import 'package:mobile/features/announcement/services/announcement_service.dart';
import 'package:mobile/features/announcement/widgets/announcement_dialog.dart';
import 'package:mobile/features/update/models/app_release.dart';
import 'package:mobile/features/update/services/update_service.dart';
import 'package:shared_preferences/shared_preferences.dart';

class _FailingAnnouncementService extends AnnouncementService {
  _FailingAnnouncementService(super.api);

  int calls = 0;

  @override
  Future<List<Announcement>> getActive() async {
    calls++;
    throw StateError('network unavailable');
  }
}

ApiClient _apiClient() {
  return ApiClient(
    baseUrl: 'https://example.com',
    getAccessToken: () => null,
    refreshSession: () async => null,
    onSessionInvalid: () {},
  );
}

void main() {
  const forcedRelease = AppRelease(
    hasUpdate: true,
    forceUpdate: true,
    latestVersion: AppReleaseVersion(versionName: '2.0.0', versionCode: 2),
    downloadUrl: 'https://example.com/app.apk',
  );
  const optionalRelease = AppRelease(
    hasUpdate: true,
    latestVersion: AppReleaseVersion(versionName: '2.0.0', versionCode: 2),
    downloadUrl: 'https://example.com/app.apk',
  );
  const importantAnnouncement = Announcement(
    id: 'important',
    title: 'Important',
    content: 'Content',
    type: 'maintenance',
    displayMode: 'startup_modal',
    target: 'android',
    priority: 10,
    dismissible: true,
  );
  const normalAnnouncement = Announcement(
    id: 'normal',
    title: 'Normal',
    content: 'Content',
    type: 'info',
    displayMode: 'startup_modal',
    target: 'all',
    priority: 20,
    dismissible: true,
  );

  test('forced update wins and ignores ignored version', () {
    final selection = StartupPromptCoordinator.selectPrompt(
      release: forcedRelease,
      announcements: const [importantAnnouncement],
      dismissedAnnouncementIds: const {},
      ignoredVersionCode: 2,
      platform: 'android',
    );

    expect(selection?.kind, StartupPromptKind.forcedUpdate);
  });

  test('important announcement wins over optional update', () {
    final selection = StartupPromptCoordinator.selectPrompt(
      release: optionalRelease,
      announcements: const [normalAnnouncement, importantAnnouncement],
      dismissedAnnouncementIds: const {},
      ignoredVersionCode: null,
      platform: 'android',
    );

    expect(selection?.kind, StartupPromptKind.importantAnnouncement);
    expect(selection?.announcement?.id, 'important');
  });

  test('ignored optional update falls through to one normal announcement', () {
    final selection = StartupPromptCoordinator.selectPrompt(
      release: optionalRelease,
      announcements: const [normalAnnouncement],
      dismissedAnnouncementIds: const {},
      ignoredVersionCode: 2,
      platform: 'android',
    );

    expect(selection?.kind, StartupPromptKind.announcement);
    expect(selection?.announcement?.id, 'normal');
  });

  test('cold start always checks despite a recent persisted check', () {
    final now = DateTime(2026, 9, 8, 12);

    expect(
      StartupPromptCoordinator.shouldCheck(
        trigger: StartupCheckTrigger.coldStart,
        lastCheckMilliseconds: now.millisecondsSinceEpoch,
        now: now,
      ),
      isTrue,
    );
  });

  test('resume checks remain throttled for six hours', () {
    final now = DateTime(2026, 9, 8, 12);

    expect(
      StartupPromptCoordinator.shouldCheck(
        trigger: StartupCheckTrigger.resume,
        lastCheckMilliseconds: now
            .subtract(const Duration(hours: 5))
            .millisecondsSinceEpoch,
        now: now,
      ),
      isFalse,
    );
    expect(
      StartupPromptCoordinator.shouldCheck(
        trigger: StartupCheckTrigger.resume,
        lastCheckMilliseconds: now
            .subtract(const Duration(hours: 6))
            .millisecondsSinceEpoch,
        now: now,
      ),
      isTrue,
    );
  });

  testWidgets(
    'cold start bypasses persisted throttle and failed fetch stays non-blocking',
    (tester) async {
      final now = DateTime(2026, 9, 8, 12);
      SharedPreferences.setMockInitialValues({
        StartupPromptCoordinator.lastCheckKey: now
            .subtract(const Duration(minutes: 5))
            .millisecondsSinceEpoch,
      });
      final preferences = await SharedPreferences.getInstance();
      final announcements = _FailingAnnouncementService(_apiClient());
      final coordinator = StartupPromptCoordinator(
        updateService: UpdateService(_apiClient()),
        announcementService: announcements,
        preferences: () async => preferences,
        packageInfo: () async => throw UnimplementedError(),
        isAndroid: false,
        platform: 'android',
        now: () => now,
      );
      late BuildContext context;
      await tester.pumpWidget(
        MaterialApp(
          home: Builder(
            builder: (value) {
              context = value;
              return const SizedBox();
            },
          ),
        ),
      );

      await coordinator.check(context, trigger: StartupCheckTrigger.coldStart);
      expect(announcements.calls, 1);
      expect(
        preferences.getInt(StartupPromptCoordinator.lastCheckKey),
        now.millisecondsSinceEpoch,
      );

      await coordinator.check(context, trigger: StartupCheckTrigger.resume);
      expect(announcements.calls, 1);
    },
  );

  testWidgets('barrier dismissal persists a dismissible announcement', (
    tester,
  ) async {
    var persisted = 0;
    await tester.pumpWidget(
      MaterialApp(
        home: Builder(
          builder: (context) => FilledButton(
            onPressed: () async {
              await showAnnouncementDialog(
                context,
                normalAnnouncement,
                onAcknowledge: () async => persisted++,
              );
            },
            child: const Text('Show'),
          ),
        ),
      ),
    );

    await tester.tap(find.text('Show'));
    await tester.pumpAndSettle();
    await tester.tapAt(const Offset(4, 4));
    await tester.pumpAndSettle();

    expect(persisted, 1);
    expect(find.text(normalAnnouncement.title), findsNothing);
  });

  testWidgets('back dismissal persists a dismissible announcement', (
    tester,
  ) async {
    var persisted = 0;
    await tester.pumpWidget(
      MaterialApp(
        home: Builder(
          builder: (context) => FilledButton(
            onPressed: () async {
              await showAnnouncementDialog(
                context,
                normalAnnouncement,
                onAcknowledge: () async => persisted++,
              );
            },
            child: const Text('Show'),
          ),
        ),
      ),
    );

    await tester.tap(find.text('Show'));
    await tester.pumpAndSettle();
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();

    expect(persisted, 1);
    expect(find.text(normalAnnouncement.title), findsNothing);
  });
}
