import 'galgame_models.dart';

class NovelStatistics {
  const NovelStatistics({this.volumeCount, this.resourceCount});

  factory NovelStatistics.fromMap(Map<String, dynamic> map) => NovelStatistics(
        volumeCount: (map['volume_count'] as num?)?.toInt(),
        resourceCount: (map['resource_count'] as num?)?.toInt(),
      );

  final int? volumeCount;
  final int? resourceCount;
}

class NovelListItem {
  const NovelListItem({
    this.id,
    this.title,
    this.originalTitle,
    this.slug,
    this.coverUrl,
    this.isCoverSensitive = false,
    this.ageRating,
    this.status,
    this.author,
    this.publisher,
    this.label,
    this.language,
    this.releaseStatus,
    this.firstReleaseDate,
    this.tags = const [],
    this.statistics,
    this.updatedAt,
    this.createdAt,
  });

  factory NovelListItem.fromMap(Map<String, dynamic> map) => NovelListItem(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        originalTitle: map['original_title'] as String?,
        slug: map['slug'] as String?,
        coverUrl: map['cover_url'] as String?,
        isCoverSensitive: map['is_cover_sensitive'] as bool? ?? false,
        ageRating: (map['age_rating'] as num?)?.toInt(),
        status: (map['status'] as num?)?.toInt(),
        author: map['author'] as String?,
        publisher: map['publisher'] as String?,
        label: map['label'] as String?,
        language: map['language'] as String?,
        releaseStatus: map['release_status'] as String?,
        firstReleaseDate: map['first_release_date'] as String?,
    tags:
        (map['tags'] as List?)
                ?.whereType<Map>()
                .map((e) => TagSummary.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        statistics: map['statistics'] is Map
        ? NovelStatistics.fromMap(Map<String, dynamic>.from(map['statistics']))
            : null,
        updatedAt: map['updated_at'] as String?,
        createdAt: map['created_at'] as String?,
      );

  final int? id;
  final String? title;
  final String? originalTitle;
  final String? slug;
  final String? coverUrl;
  final bool isCoverSensitive;
  final int? ageRating;
  final int? status;
  final String? author;
  final String? publisher;
  final String? label;
  final String? language;
  final String? releaseStatus;
  final String? firstReleaseDate;
  final List<TagSummary> tags;
  final NovelStatistics? statistics;
  final String? updatedAt;
  final String? createdAt;
}

class RelatedWorkData {
  const RelatedWorkData({
    this.workId,
    this.title,
    this.originalTitle,
    this.slug,
    this.coverUrl,
    this.coverSensitive = false,
    this.ageRating,
    this.relationType,
    this.relationId,
  });

  factory RelatedWorkData.fromMap(Map<String, dynamic> map) => RelatedWorkData(
        workId: (map['work_id'] as num?)?.toInt(),
        title: map['title'] as String?,
    originalTitle: map['original_title'] as String?,
    slug: map['slug'] as String?,
        coverUrl: map['cover_url'] as String?,
    coverSensitive: map['cover_sensitive'] as bool? ?? false,
    ageRating: (map['age_rating'] as num?)?.toInt(),
        relationType: map['relation_type'] as String?,
        relationId: (map['relation_id'] as num?)?.toInt(),
      );

  final int? workId;
  final String? title;
  final String? originalTitle;
  final String? slug;
  final String? coverUrl;
  final bool coverSensitive;
  final int? ageRating;
  final String? relationType;
  final int? relationId;
}

class VolumeSummary {
  const VolumeSummary({
    this.id,
    this.title,
    this.originalTitle,
    this.volumeNumber,
    this.isbn,
    this.coverUrl,
    this.releaseDate,
  });

  factory VolumeSummary.fromMap(Map<String, dynamic> map) => VolumeSummary(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
    originalTitle: map['original_title'] as String?,
        volumeNumber: (map['volume_number'] as num?)?.toInt(),
        isbn: map['isbn'] as String?,
        coverUrl: map['cover_url'] as String?,
        releaseDate: map['release_date'] as String?,
      );

  final int? id;
  final String? title;
  final String? originalTitle;
  final int? volumeNumber;
  final String? isbn;
  final String? coverUrl;
  final String? releaseDate;
}

class NovelDetail extends NovelListItem {
  const NovelDetail({
    super.id,
    super.title,
    super.originalTitle,
    super.slug,
    super.coverUrl,
    super.isCoverSensitive = false,
    super.ageRating,
    super.status,
    super.author,
    super.publisher,
    super.label,
    super.language,
    super.releaseStatus,
    super.firstReleaseDate,
    super.tags = const [],
    super.statistics,
    super.updatedAt,
    super.createdAt,
    this.illustrator,
    this.region,
    this.summary,
    this.officialWebsite,
    this.contributorCount,
    this.contributors,
    this.relatedGalgames,
    this.volumes,
    this.rejectReason,
  });

  factory NovelDetail.fromMap(Map<String, dynamic> map) => NovelDetail(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        originalTitle: map['original_title'] as String?,
        slug: map['slug'] as String?,
        coverUrl: map['cover_url'] as String?,
        isCoverSensitive: map['is_cover_sensitive'] as bool? ?? false,
        ageRating: (map['age_rating'] as num?)?.toInt(),
        status: (map['status'] as num?)?.toInt(),
        author: map['author'] as String?,
        publisher: map['publisher'] as String?,
        label: map['label'] as String?,
        language: map['language'] as String?,
        releaseStatus: map['release_status'] as String?,
        firstReleaseDate: map['first_release_date'] as String?,
    tags:
        (map['tags'] as List?)
                ?.whereType<Map>()
                .map((e) => TagSummary.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        statistics: map['statistics'] is Map
        ? NovelStatistics.fromMap(Map<String, dynamic>.from(map['statistics']))
            : null,
        updatedAt: map['updated_at'] as String?,
        createdAt: map['created_at'] as String?,
        illustrator: map['illustrator'] as String?,
        region: map['region'] as String?,
        summary: map['summary'] as String?,
        officialWebsite: map['official_website'] as String?,
        contributorCount: (map['contributor_count'] as num?)?.toInt(),
    contributors:
        (map['contributors'] as List?)
                ?.whereType<Map>()
                .map((e) => ContributorData.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
    relatedGalgames:
        (map['related_galgames'] as List?)
                ?.whereType<Map>()
                .map((e) => RelatedWorkData.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
    volumes:
        (map['volumes'] as List?)
                ?.whereType<Map>()
                .map((e) => VolumeSummary.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        rejectReason: map['reject_reason'] as String?,
      );

  final String? illustrator;
  final String? region;
  final String? summary;
  final String? officialWebsite;
  final int? contributorCount;
  final List<ContributorData>? contributors;
  final List<RelatedWorkData>? relatedGalgames;
  final List<VolumeSummary>? volumes;
  final String? rejectReason;
}

class VolumeData {
  const VolumeData({
    this.id,
    this.novelId,
    this.title,
    this.originalTitle,
    this.volumeNumber,
    this.isbn,
    this.coverUrl,
    this.releaseDate,
    this.summary,
    this.status,
    this.sortOrder,
    this.rejectReason,
  });

  factory VolumeData.fromMap(Map<String, dynamic> map) => VolumeData(
        id: (map['id'] as num?)?.toInt(),
        novelId: (map['novel_id'] as num?)?.toInt(),
        title: map['title'] as String?,
        originalTitle: map['original_title'] as String?,
        volumeNumber: (map['volume_number'] as num?)?.toInt(),
        isbn: map['isbn'] as String?,
        coverUrl: map['cover_url'] as String?,
        releaseDate: map['release_date'] as String?,
        summary: map['summary'] as String?,
        status: (map['status'] as num?)?.toInt(),
        sortOrder: (map['sort_order'] as num?)?.toInt(),
        rejectReason: map['reject_reason'] as String?,
      );

  final int? id;
  final int? novelId;
  final String? title;
  final String? originalTitle;
  final int? volumeNumber;
  final String? isbn;
  final String? coverUrl;
  final String? releaseDate;
  final String? summary;
  final int? status;
  final int? sortOrder;
  final String? rejectReason;
}
