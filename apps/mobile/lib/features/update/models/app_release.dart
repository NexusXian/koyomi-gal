class AppRelease {
  const AppRelease({
    required this.hasUpdate,
    this.forceUpdate = false,
    this.latestVersion,
    this.title,
    this.changelog,
    this.downloadUrl,
    this.fileSize,
    this.sha256,
    this.publishedAt,
  });

  factory AppRelease.fromMap(Map<String, dynamic> map) {
    final latestVersion = map['latestVersion'];
    return AppRelease(
      hasUpdate: map['hasUpdate'] as bool? ?? false,
      forceUpdate: map['forceUpdate'] as bool? ?? false,
      latestVersion: latestVersion is Map
          ? AppReleaseVersion.fromMap(Map<String, dynamic>.from(latestVersion))
          : null,
      title: map['title'] as String?,
      changelog: map['changelog'] as String?,
      downloadUrl: map['downloadUrl'] as String?,
      fileSize: (map['fileSize'] as num?)?.toInt(),
      sha256: map['sha256'] as String?,
      publishedAt: DateTime.tryParse(map['publishedAt'] as String? ?? ''),
    );
  }

  final bool hasUpdate;
  final bool forceUpdate;
  final AppReleaseVersion? latestVersion;
  final String? title;
  final String? changelog;
  final String? downloadUrl;
  final int? fileSize;
  final String? sha256;
  final DateTime? publishedAt;
}

class AppReleaseVersion {
  const AppReleaseVersion({
    required this.versionName,
    required this.versionCode,
  });

  factory AppReleaseVersion.fromMap(Map<String, dynamic> map) {
    return AppReleaseVersion(
      versionName: map['versionName'] as String? ?? '',
      versionCode: (map['versionCode'] as num?)?.toInt() ?? 0,
    );
  }

  final String versionName;
  final int versionCode;
}
