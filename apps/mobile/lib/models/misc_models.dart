import 'galgame_models.dart';
import 'post_models.dart';

class HomeBanner {
  const HomeBanner({
    this.id,
    this.title,
    this.subtitle,
    this.imageUrl,
    this.linkType,
    this.linkValue,
  });

  factory HomeBanner.fromMap(Map<String, dynamic> map) => HomeBanner(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        subtitle: map['subtitle'] as String?,
        imageUrl: map['image_url'] as String?,
        linkType: map['link_type'] as String?,
        linkValue: map['link_value'] as String?,
      );

  final int? id;
  final String? title;
  final String? subtitle;
  final String? imageUrl;
  final String? linkType;
  final String? linkValue;
}

class AnnouncementData {
  const AnnouncementData({
    this.id,
    this.title,
    this.summary,
    this.coverUrl,
    this.isPinned = false,
    this.publishedAt,
    this.type,
  });

  factory AnnouncementData.fromMap(Map<String, dynamic> map) =>
      AnnouncementData(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        summary: map['summary'] as String?,
        coverUrl: map['cover_url'] as String?,
        isPinned: map['is_pinned'] as bool? ?? false,
        publishedAt: map['published_at'] as String?,
        type: map['type'] as String?,
      );

  final int? id;
  final String? title;
  final String? summary;
  final String? coverUrl;
  final bool isPinned;
  final String? publishedAt;
  final String? type;
}

class HomeData {
  const HomeData({
    this.banners = const [],
    this.announcements = const [],
    this.latestGalgames = const [],
    this.popularGalgames = const [],
    this.latestPosts = const [],
    this.popularPosts = const [],
  });

  factory HomeData.fromMap(Map<String, dynamic> map) => HomeData(
        banners: (map['banners'] as List?)
                ?.whereType<Map>()
                .map((e) => HomeBanner.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        announcements: (map['announcements'] as List?)
                ?.whereType<Map>()
                .map((e) => AnnouncementData.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        latestGalgames: (map['latest_galgames'] as List?)
                ?.whereType<Map>()
                .map((e) => HomeGalgame.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        popularGalgames: (map['popular_galgames'] as List?)
                ?.whereType<Map>()
                .map((e) => HomeGalgame.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        latestPosts: (map['latest_posts'] as List?)
                ?.whereType<Map>()
                .map((e) => HomePost.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        popularPosts: (map['popular_posts'] as List?)
                ?.whereType<Map>()
                .map((e) => HomePost.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
      );

  final List<HomeBanner> banners;
  final List<AnnouncementData> announcements;
  final List<HomeGalgame> latestGalgames;
  final List<HomeGalgame> popularGalgames;
  final List<HomePost> latestPosts;
  final List<HomePost> popularPosts;
}

class TagData {
  const TagData({this.id, this.name, this.slug, this.description});

  factory TagData.fromMap(Map<String, dynamic> map) => TagData(
        id: (map['id'] as num?)?.toInt(),
        name: map['name'] as String?,
        slug: map['slug'] as String?,
        description: map['description'] as String?,
      );

  final int? id;
  final String? name;
  final String? slug;
  final String? description;
}

class DeveloperData {
  const DeveloperData({
    this.id,
    this.name,
    this.originalName,
    this.slug,
    this.description,
    this.logoUrl,
    this.website,
  });

  factory DeveloperData.fromMap(Map<String, dynamic> map) => DeveloperData(
        id: (map['id'] as num?)?.toInt(),
        name: map['name'] as String?,
        originalName: map['original_name'] as String?,
        slug: map['slug'] as String?,
        description: map['description'] as String?,
        logoUrl: map['logo_url'] as String?,
        website: map['website'] as String?,
      );

  final int? id;
  final String? name;
  final String? originalName;
  final String? slug;
  final String? description;
  final String? logoUrl;
  final String? website;
}

class ResourceLinkData {
  const ResourceLinkData({this.id, this.url});

  factory ResourceLinkData.fromMap(Map<String, dynamic> map) =>
      ResourceLinkData(
        id: (map['id'] as num?)?.toInt(),
        url: map['url'] as String?,
      );

  final int? id;
  final String? url;
}

class ResourceData {
  const ResourceData({
    this.id,
    this.title,
    this.description,
    this.type,
    this.status,
    this.targetId,
    this.targetType,
    this.links = const [],
    this.uploaderId,
    this.createdAt,
  });

  factory ResourceData.fromMap(Map<String, dynamic> map) => ResourceData(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        description: map['description'] as String?,
        type: (map['type'] as num?)?.toInt(),
        status: (map['status'] as num?)?.toInt(),
        targetId: (map['target_id'] as num?)?.toInt(),
        targetType: map['target_type'] as String?,
        links: (map['links'] as List?)
                ?.whereType<Map>()
                .map((e) => ResourceLinkData.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        uploaderId: (map['uploader_id'] as num?)?.toInt(),
        createdAt: map['created_at'] as String?,
      );

  final int? id;
  final String? title;
  final String? description;
  final int? type;
  final int? status;
  final int? targetId;
  final String? targetType;
  final List<ResourceLinkData> links;
  final int? uploaderId;
  final String? createdAt;
}

class ImageAsset {
  const ImageAsset({
    this.id,
    this.url,
    this.width,
    this.height,
    this.mimeType,
    this.size,
    this.category,
    this.status,
  });

  factory ImageAsset.fromMap(Map<String, dynamic> map) => ImageAsset(
        id: (map['id'] as num?)?.toInt(),
        url: map['url'] as String?,
        width: (map['width'] as num?)?.toInt(),
        height: (map['height'] as num?)?.toInt(),
        mimeType: map['mime_type'] as String?,
        size: (map['size'] as num?)?.toInt(),
        category: map['category'] as String?,
        status: (map['status'] as num?)?.toInt(),
      );

  final int? id;
  final String? url;
  final int? width;
  final int? height;
  final String? mimeType;
  final int? size;
  final String? category;
  final int? status;
}
