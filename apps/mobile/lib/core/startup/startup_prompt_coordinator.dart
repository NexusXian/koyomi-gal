import 'package:flutter/material.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../features/announcement/models/announcement.dart';
import '../../features/announcement/services/announcement_service.dart';
import '../../features/announcement/widgets/announcement_dialog.dart';
import '../../features/update/models/app_release.dart';
import '../../features/update/services/update_service.dart';
import '../../features/update/widgets/update_dialog.dart';

enum StartupPromptKind {
  forcedUpdate,
  importantAnnouncement,
  optionalUpdate,
  announcement,
}

enum StartupCheckTrigger { coldStart, resume }

class StartupPromptSelection {
  const StartupPromptSelection._({
    required this.kind,
    this.release,
    this.announcement,
  });

  final StartupPromptKind kind;
  final AppRelease? release;
  final Announcement? announcement;
}

class StartupPromptCoordinator {
  StartupPromptCoordinator({
    required this._updateService,
    required this._announcementService,
    required this._preferences,
    required this._packageInfo,
    required this.isAndroid,
    required this.platform,
    DateTime Function()? now,
  }) : _now = now ?? DateTime.now;

  static const checkInterval = Duration(hours: 6);
  static const lastCheckKey = 'last_update_check_at';
  static const ignoredVersionKey = 'ignored_version_code';
  static const dismissedAnnouncementsKey = 'dismissed_announcement_ids';

  final UpdateService _updateService;
  final AnnouncementService _announcementService;
  final Future<SharedPreferences> Function() _preferences;
  final Future<PackageInfo> Function() _packageInfo;
  final DateTime Function() _now;
  final bool isAndroid;
  final String platform;

  Future<void>? _inFlight;
  bool _dialogVisible = false;

  Future<void> check(
    BuildContext context, {
    required StartupCheckTrigger trigger,
  }) {
    final current = _inFlight;
    if (current != null) {
      return current;
    }
    final future = _run(context, trigger: trigger);
    _inFlight = future;
    future.whenComplete(() {
      if (identical(_inFlight, future)) {
        _inFlight = null;
      }
    });
    return future;
  }

  Future<void> _run(
    BuildContext context, {
    required StartupCheckTrigger trigger,
  }) async {
    if (_dialogVisible) {
      return;
    }

    try {
      final preferences = await _preferences();
      final now = _now();
      if (!shouldCheck(
        trigger: trigger,
        lastCheckMilliseconds: preferences.getInt(lastCheckKey),
        now: now,
      )) {
        return;
      }
      await preferences.setInt(lastCheckKey, now.millisecondsSinceEpoch);

      AppRelease? release;
      var announcements = <Announcement>[];

      Future<void> loadRelease() async {
        if (!isAndroid) {
          return;
        }
        try {
          final info = await _packageInfo();
          release = await _updateService.getLatestRelease(
            versionCode: int.tryParse(info.buildNumber) ?? 0,
            versionName: info.version,
          );
        } catch (_) {}
      }

      Future<void> loadAnnouncements() async {
        try {
          announcements = await _announcementService.getActive();
        } catch (_) {}
      }

      await Future.wait([loadRelease(), loadAnnouncements()]);
      if (!context.mounted) {
        return;
      }

      final selection = selectPrompt(
        release: release,
        announcements: announcements,
        dismissedAnnouncementIds:
            preferences.getStringList(dismissedAnnouncementsKey)?.toSet() ??
            const {},
        ignoredVersionCode: preferences.getInt(ignoredVersionKey),
        platform: platform,
      );
      if (selection == null) {
        return;
      }

      _dialogVisible = true;
      try {
        switch (selection.kind) {
          case StartupPromptKind.forcedUpdate:
          case StartupPromptKind.optionalUpdate:
            final selectedRelease = selection.release!;
            final result = await showUpdateDialog(context, selectedRelease);
            if (result == UpdateDialogResult.ignore &&
                !selectedRelease.forceUpdate) {
              await preferences.setInt(
                ignoredVersionKey,
                selectedRelease.latestVersion!.versionCode,
              );
            }
          case StartupPromptKind.importantAnnouncement:
          case StartupPromptKind.announcement:
            final announcement = selection.announcement!;
            await showAnnouncementDialog(
              context,
              announcement,
              onAcknowledge: () async {
                final dismissed =
                    preferences
                        .getStringList(dismissedAnnouncementsKey)
                        ?.toSet() ??
                    <String>{};
                dismissed.add(announcement.id);
                await preferences.setStringList(
                  dismissedAnnouncementsKey,
                  dismissed.toList(),
                );
              },
            );
        }
      } finally {
        _dialogVisible = false;
      }
    } catch (_) {}
  }

  static bool isCheckDue({
    required int? lastCheckMilliseconds,
    required DateTime now,
  }) {
    if (lastCheckMilliseconds == null) {
      return true;
    }
    final lastCheck = DateTime.fromMillisecondsSinceEpoch(
      lastCheckMilliseconds,
    );
    return now.difference(lastCheck) >= checkInterval;
  }

  static bool shouldCheck({
    required StartupCheckTrigger trigger,
    required int? lastCheckMilliseconds,
    required DateTime now,
  }) {
    return trigger == StartupCheckTrigger.coldStart ||
        isCheckDue(lastCheckMilliseconds: lastCheckMilliseconds, now: now);
  }

  static StartupPromptSelection? selectPrompt({
    required AppRelease? release,
    required List<Announcement> announcements,
    required Set<String> dismissedAnnouncementIds,
    required int? ignoredVersionCode,
    required String platform,
  }) {
    final hasUsableUpdate =
        release?.hasUpdate == true &&
        release?.latestVersion != null &&
        (release?.downloadUrl?.isNotEmpty ?? false);
    if (hasUsableUpdate && release!.forceUpdate) {
      return StartupPromptSelection._(
        kind: StartupPromptKind.forcedUpdate,
        release: release,
      );
    }

    final eligibleAnnouncements =
        announcements.where((announcement) {
          return announcement.id.isNotEmpty &&
              announcement.displayMode == 'startup_modal' &&
              announcement.targetsPlatform(platform) &&
              !dismissedAnnouncementIds.contains(announcement.id);
        }).toList()..sort((left, right) {
          final priority = right.priority.compareTo(left.priority);
          if (priority != 0) {
            return priority;
          }
          final updated =
              (right.updatedAt ?? DateTime.fromMillisecondsSinceEpoch(0))
                  .compareTo(
                    left.updatedAt ?? DateTime.fromMillisecondsSinceEpoch(0),
                  );
          return updated != 0 ? updated : left.id.compareTo(right.id);
        });

    for (final announcement in eligibleAnnouncements) {
      if (announcement.isImportant) {
        return StartupPromptSelection._(
          kind: StartupPromptKind.importantAnnouncement,
          announcement: announcement,
        );
      }
    }

    if (hasUsableUpdate &&
        release!.latestVersion!.versionCode != ignoredVersionCode) {
      return StartupPromptSelection._(
        kind: StartupPromptKind.optionalUpdate,
        release: release,
      );
    }

    for (final announcement in eligibleAnnouncements) {
      if (!announcement.isImportant) {
        return StartupPromptSelection._(
          kind: StartupPromptKind.announcement,
          announcement: announcement,
        );
      }
    }
    return null;
  }
}
