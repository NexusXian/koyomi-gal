class DeveloperSummary {
  const DeveloperSummary({this.id, this.name});

  factory DeveloperSummary.fromMap(Map<String, dynamic> map) =>
      DeveloperSummary(
        id: (map['id'] as num?)?.toInt(),
        name: map['name'] as String?,
      );

  final int? id;
  final String? name;
}

class TagSummary {
  const TagSummary({this.id, this.name});

  factory TagSummary.fromMap(Map<String, dynamic> map) => TagSummary(
        id: (map['id'] as num?)?.toInt(),
        name: map['name'] as String?,
      );

  final int? id;
  final String? name;
}

class RatingSummary {
  const RatingSummary({this.average, this.count});

  factory RatingSummary.fromMap(Map<String, dynamic> map) => RatingSummary(
        average: (map['average'] as num?)?.toDouble(),
        count: (map['count'] as num?)?.toInt(),
      );

  final double? average;
  final int? count;
}

class GalgameStatistics {
  const GalgameStatistics({
    this.favoriteCount,
    this.postCount,
    this.resourceCount,
  });

  factory GalgameStatistics.fromMap(Map<String, dynamic> map) =>
      GalgameStatistics(
        favoriteCount: (map['favorite_count'] as num?)?.toInt(),
        postCount: (map['post_count'] as num?)?.toInt(),
        resourceCount: (map['resource_count'] as num?)?.toInt(),
      );

  final int? favoriteCount;
  final int? postCount;
  final int? resourceCount;
}

class ContributorData {
  const ContributorData({
    this.userId,
    this.username,
    this.avatarUrl,
    this.contributionCount,
    this.lastContributedAt,
  });

  factory ContributorData.fromMap(Map<String, dynamic> map) => ContributorData(
        userId: (map['user_id'] as num?)?.toInt(),
        username: map['username'] as String?,
        avatarUrl: map['avatar_url'] as String?,
        contributionCount: (map['contribution_count'] as num?)?.toInt(),
        lastContributedAt: map['last_contributed_at'] as String?,
      );

  final int? userId;
  final String? username;
  final String? avatarUrl;
  final int? contributionCount;
  final String? lastContributedAt;
}

class GalgameListItem {
  const GalgameListItem({
    this.id,
    this.title,
    this.originalTitle,
    this.romajiTitle,
    this.slug,
    this.coverUrl,
    this.coverSensitive = false,
    this.ageRating,
    this.status,
    this.releaseDate,
    this.developer,
    this.rating,
    this.tags = const [],
    this.statistics,
  });

  factory GalgameListItem.fromMap(Map<String, dynamic> map) => GalgameListItem(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        originalTitle: map['original_title'] as String?,
        romajiTitle: map['romaji_title'] as String?,
        slug: map['slug'] as String?,
        coverUrl: map['cover_url'] as String?,
        coverSensitive: map['cover_sensitive'] as bool? ?? false,
        ageRating: (map['age_rating'] as num?)?.toInt(),
        status: (map['status'] as num?)?.toInt(),
        releaseDate: map['release_date'] as String?,
        developer: map['developer'] is Map
            ? DeveloperSummary.fromMap(
                Map<String, dynamic>.from(map['developer']))
            : null,
        rating: map['rating'] is Map
            ? RatingSummary.fromMap(Map<String, dynamic>.from(map['rating']))
            : null,
        tags: (map['tags'] as List?)
                ?.whereType<Map>()
                .map((e) => TagSummary.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        statistics: map['statistics'] is Map
            ? GalgameStatistics.fromMap(
                Map<String, dynamic>.from(map['statistics']))
            : null,
      );

  final int? id;
  final String? title;
  final String? originalTitle;
  final String? romajiTitle;
  final String? slug;
  final String? coverUrl;
  final bool coverSensitive;
  final int? ageRating;
  final int? status;
  final String? releaseDate;
  final DeveloperSummary? developer;
  final RatingSummary? rating;
  final List<TagSummary> tags;
  final GalgameStatistics? statistics;
}

class RelatedNovelData {
  const RelatedNovelData({
    this.workId,
    this.title,
    this.coverUrl,
    this.relationType,
    this.relationId,
  });

  factory RelatedNovelData.fromMap(Map<String, dynamic> map) =>
      RelatedNovelData(
        workId: (map['work_id'] as num?)?.toInt(),
        title: map['title'] as String?,
        coverUrl: map['cover_url'] as String?,
        relationType: map['relation_type'] as String?,
        relationId: (map['relation_id'] as num?)?.toInt(),
      );

  final int? workId;
  final String? title;
  final String? coverUrl;
  final String? relationType;
  final int? relationId;
}

class GalgameDetail extends GalgameListItem {
  const GalgameDetail({
    super.id,
    super.title,
    super.originalTitle,
    super.romajiTitle,
    super.slug,
    super.coverUrl,
    super.coverSensitive = false,
    super.ageRating,
    super.status,
    super.releaseDate,
    super.developer,
    super.rating,
    super.tags = const [],
    super.statistics,
    this.aliases,
    this.bannerUrl,
    this.description,
    this.descriptionSource,
    this.contributorCount,
    this.contributors,
    this.relatedNovels,
    this.createdAt,
    this.updatedAt,
  });

  factory GalgameDetail.fromMap(Map<String, dynamic> map) => GalgameDetail(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        originalTitle: map['original_title'] as String?,
        romajiTitle: map['romaji_title'] as String?,
        slug: map['slug'] as String?,
        coverUrl: map['cover_url'] as String?,
        coverSensitive: map['cover_sensitive'] as bool? ?? false,
        ageRating: (map['age_rating'] as num?)?.toInt(),
        status: (map['status'] as num?)?.toInt(),
        releaseDate: map['release_date'] as String?,
        developer: map['developer'] is Map
            ? DeveloperSummary.fromMap(
                Map<String, dynamic>.from(map['developer']))
            : null,
        rating: map['rating'] is Map
            ? RatingSummary.fromMap(Map<String, dynamic>.from(map['rating']))
            : null,
        tags: (map['tags'] as List?)
                ?.whereType<Map>()
                .map((e) => TagSummary.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        statistics: map['statistics'] is Map
            ? GalgameStatistics.fromMap(
                Map<String, dynamic>.from(map['statistics']))
            : null,
        aliases: (map['aliases'] as List?)?.map((e) => e.toString()).toList(),
        bannerUrl: map['banner_url'] as String?,
        description: map['description'] as String?,
        descriptionSource: map['description_source'] as String?,
        contributorCount: (map['contributor_count'] as num?)?.toInt(),
        contributors: (map['contributors'] as List?)
                ?.whereType<Map>()
                .map((e) => ContributorData.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        relatedNovels: (map['related_novels'] as List?)
                ?.whereType<Map>()
                .map((e) => RelatedNovelData.fromMap(Map<String, dynamic>.from(e)))
                .toList() ??
            const [],
        createdAt: map['created_at'] as String?,
        updatedAt: map['updated_at'] as String?,
      );

  final List<String>? aliases;
  final String? bannerUrl;
  final String? description;
  final String? descriptionSource;
  final int? contributorCount;
  final List<ContributorData>? contributors;
  final List<RelatedNovelData>? relatedNovels;
  final String? createdAt;
  final String? updatedAt;
}

class GalgameCharacter {
  const GalgameCharacter({
    this.id,
    this.characterId,
    this.name,
    this.originalName,
    this.imageUrl,
    this.description,
    this.publicDescription,
    this.role,
    this.gender,
    this.birthday,
    this.bloodType,
    this.height,
    this.hasSpoiler = false,
  });

  factory GalgameCharacter.fromMap(Map<String, dynamic> map) =>
      GalgameCharacter(
        id: (map['id'] as num?)?.toInt(),
        characterId: (map['character_id'] as num?)?.toInt(),
        name: map['name'] as String?,
        originalName: map['original_name'] as String?,
        imageUrl: map['image_url'] as String?,
        description: map['description'] as String?,
        publicDescription: map['public_description'] as String?,
        role: map['role'] as String?,
        gender: map['gender'] as String?,
        birthday: map['birthday'] as String?,
        bloodType: map['blood_type'] as String?,
        height: (map['height'] as num?)?.toInt(),
        hasSpoiler: map['has_spoiler'] as bool? ?? false,
      );

  final int? id;
  final int? characterId;
  final String? name;
  final String? originalName;
  final String? imageUrl;
  final String? description;
  final String? publicDescription;
  final String? role;
  final String? gender;
  final String? birthday;
  final String? bloodType;
  final int? height;
  final bool hasSpoiler;
}

class GalleryImage {
  const GalleryImage({
    this.id,
    this.url,
    this.title,
    this.description,
    this.imageType,
    this.isSpoiler = false,
    this.width,
    this.height,
  });

  factory GalleryImage.fromMap(Map<String, dynamic> map) => GalleryImage(
        id: (map['id'] as num?)?.toInt(),
        url: map['url'] as String?,
        title: map['title'] as String?,
        description: map['description'] as String?,
        imageType: (map['image_type'] as num?)?.toInt(),
        isSpoiler: map['is_spoiler'] as bool? ?? false,
        width: (map['width'] as num?)?.toInt(),
        height: (map['height'] as num?)?.toInt(),
      );

  final int? id;
  final String? url;
  final String? title;
  final String? description;
  final int? imageType;
  final bool isSpoiler;
  final int? width;
  final int? height;
}

class FavoriteData {
  const FavoriteData({this.favorited, this.createdAt});

  factory FavoriteData.fromMap(Map<String, dynamic> map) => FavoriteData(
        favorited: map['favorited'] as bool?,
        createdAt: map['created_at'] as String?,
      );

  final bool? favorited;
  final String? createdAt;
}

class RatingData {
  const RatingData({this.score, this.createdAt, this.updatedAt});

  factory RatingData.fromMap(Map<String, dynamic> map) => RatingData(
        score: (map['score'] as num?)?.toInt(),
        createdAt: map['created_at'] as String?,
        updatedAt: map['updated_at'] as String?,
      );

  final int? score;
  final String? createdAt;
  final String? updatedAt;
}

class UserStateData {
  const UserStateData({this.state, this.playTimeMinutes, this.updatedAt});

  factory UserStateData.fromMap(Map<String, dynamic> map) => UserStateData(
        state: (map['state'] as num?)?.toInt(),
        playTimeMinutes: (map['play_time_minutes'] as num?)?.toInt(),
        updatedAt: map['updated_at'] as String?,
      );

  final int? state;
  final int? playTimeMinutes;
  final String? updatedAt;
}

class GalgameUserRelation {
  const GalgameUserRelation({
    this.galgameId,
    this.favorite,
    this.rating,
    this.state,
  });

  factory GalgameUserRelation.fromMap(Map<String, dynamic> map) =>
      GalgameUserRelation(
        galgameId: (map['galgame_id'] as num?)?.toInt(),
        favorite: map['favorite'] is Map
            ? FavoriteData.fromMap(Map<String, dynamic>.from(map['favorite']))
            : null,
        rating: map['rating'] is Map
            ? RatingData.fromMap(Map<String, dynamic>.from(map['rating']))
            : null,
        state: map['state'] is Map
            ? UserStateData.fromMap(Map<String, dynamic>.from(map['state']))
            : null,
      );

  final int? galgameId;
  final FavoriteData? favorite;
  final RatingData? rating;
  final UserStateData? state;
}

class HomeGalgame {
  const HomeGalgame({
    this.id,
    this.title,
    this.coverUrl,
    this.coverSensitive = false,
    this.developer,
    this.ratingAverage,
    this.favoriteCount,
    this.releaseDate,
  });

  factory HomeGalgame.fromMap(Map<String, dynamic> map) => HomeGalgame(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        coverUrl: map['cover_url'] as String?,
        coverSensitive: map['cover_sensitive'] as bool? ?? false,
        developer: map['developer'] is Map
            ? DeveloperSummary.fromMap(
                Map<String, dynamic>.from(map['developer']))
            : null,
        ratingAverage: (map['rating_average'] as num?)?.toDouble(),
        favoriteCount: (map['favorite_count'] as num?)?.toInt(),
        releaseDate: map['release_date'] as String?,
      );

  final int? id;
  final String? title;
  final String? coverUrl;
  final bool coverSensitive;
  final DeveloperSummary? developer;
  final double? ratingAverage;
  final int? favoriteCount;
  final String? releaseDate;
}
