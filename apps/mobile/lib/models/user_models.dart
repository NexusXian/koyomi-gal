class LevelSummary {
  const LevelSummary({this.level, this.name, this.color, this.iconUrl});

  factory LevelSummary.fromMap(Map<String, dynamic> map) => LevelSummary(
        level: (map['level'] as num?)?.toInt(),
        name: map['name'] as String?,
        color: map['color'] as String?,
        iconUrl: map['icon_url'] as String?,
      );

  final int? level;
  final String? name;
  final String? color;
  final String? iconUrl;
}

class CommunityUserSummary {
  const CommunityUserSummary({
    this.id,
    this.username,
    this.displayName,
    this.avatarUrl,
    this.level,
  });

  factory CommunityUserSummary.fromMap(Map<String, dynamic> map) =>
      CommunityUserSummary(
        id: (map['id'] as num?)?.toInt(),
        username: map['username'] as String?,
        displayName: map['display_name'] as String?,
        avatarUrl: map['avatar_url'] as String?,
        level: map['level'] is Map
            ? LevelSummary.fromMap(Map<String, dynamic>.from(map['level']))
            : null,
      );

  final int? id;
  final String? username;
  final String? displayName;
  final String? avatarUrl;
  final LevelSummary? level;
}

class ProfileAccess {
  const ProfileAccess({
    this.canViewProfile = false,
    this.canViewPosts = false,
    this.canViewComments = false,
    this.canViewRatings = false,
    this.canViewFavorites = false,
    this.canViewActivity = false,
    this.canViewBirthday = false,
    this.canViewLocation = false,
  });

  factory ProfileAccess.fromMap(Map<String, dynamic> map) => ProfileAccess(
        canViewProfile: map['can_view_profile'] as bool? ?? false,
        canViewPosts: map['can_view_posts'] as bool? ?? false,
        canViewComments: map['can_view_comments'] as bool? ?? false,
        canViewRatings: map['can_view_ratings'] as bool? ?? false,
        canViewFavorites: map['can_view_favorites'] as bool? ?? false,
        canViewActivity: map['can_view_activity'] as bool? ?? false,
        canViewBirthday: map['can_view_birthday'] as bool? ?? false,
        canViewLocation: map['can_view_location'] as bool? ?? false,
      );

  final bool canViewProfile;
  final bool canViewPosts;
  final bool canViewComments;
  final bool canViewRatings;
  final bool canViewFavorites;
  final bool canViewActivity;
  final bool canViewBirthday;
  final bool canViewLocation;
}

class PublicUserProfile {
  const PublicUserProfile({
    this.id,
    this.username,
    this.displayName,
    this.avatarUrl,
    this.bannerUrl,
    this.bio,
    this.birthday,
    this.gender,
    this.location,
    this.websiteUrl,
    this.registeredAt,
    this.postCount = 0,
    this.commentCount = 0,
    this.ratingCount = 0,
    this.favoriteCount = 0,
    this.level,
    this.isPrivate = false,
    this.isRestricted = false,
    this.isSelf = false,
    this.access,
  });

  factory PublicUserProfile.fromMap(Map<String, dynamic> map) =>
      PublicUserProfile(
        id: (map['id'] as num?)?.toInt(),
        username: map['username'] as String?,
        displayName: map['display_name'] as String?,
        avatarUrl: map['avatar_url'] as String?,
        bannerUrl: map['banner_url'] as String?,
        bio: map['bio'] as String?,
        birthday: map['birthday'] as String?,
        gender: map['gender'] as String?,
        location: map['location'] as String?,
        websiteUrl: map['website_url'] as String?,
        registeredAt: map['registered_at'] as String?,
        postCount: (map['post_count'] as num?)?.toInt() ?? 0,
        commentCount: (map['comment_count'] as num?)?.toInt() ?? 0,
        ratingCount: (map['rating_count'] as num?)?.toInt() ?? 0,
        favoriteCount: (map['favorite_count'] as num?)?.toInt() ?? 0,
        level: map['level'] is Map
            ? LevelSummary.fromMap(Map<String, dynamic>.from(map['level']))
            : null,
        isPrivate: map['is_private'] as bool? ?? false,
        isRestricted: map['is_restricted'] as bool? ?? false,
        isSelf: map['is_self'] as bool? ?? false,
        access: map['access'] is Map
            ? ProfileAccess.fromMap(Map<String, dynamic>.from(map['access']))
            : null,
      );

  final int? id;
  final String? username;
  final String? displayName;
  final String? avatarUrl;
  final String? bannerUrl;
  final String? bio;
  final String? birthday;
  final String? gender;
  final String? location;
  final String? websiteUrl;
  final String? registeredAt;
  final int postCount;
  final int commentCount;
  final int ratingCount;
  final int favoriteCount;
  final LevelSummary? level;
  final bool isPrivate;
  final bool isRestricted;
  final bool isSelf;
  final ProfileAccess? access;
}

class UserLevelData {
  const UserLevelData({
    this.level,
    this.levelName,
    this.color,
    this.totalExp,
    this.currentLevelExp,
    this.nextLevel,
    this.nextLevelName,
    this.nextLevelExp,
    this.remainingExp,
    this.progress,
    this.isMaxLevel = false,
    this.checkedInToday = false,
    this.consecutiveDays,
  });

  factory UserLevelData.fromMap(Map<String, dynamic> map) => UserLevelData(
        level: (map['level'] as num?)?.toInt(),
        levelName: map['level_name'] as String?,
        color: map['color'] as String?,
        totalExp: (map['total_exp'] as num?)?.toInt(),
        currentLevelExp: (map['current_level_exp'] as num?)?.toInt(),
        nextLevel: (map['next_level'] as num?)?.toInt(),
        nextLevelName: map['next_level_name'] as String?,
        nextLevelExp: (map['next_level_exp'] as num?)?.toInt(),
        remainingExp: (map['remaining_exp'] as num?)?.toInt(),
        progress: (map['progress'] as num?)?.toDouble(),
        isMaxLevel: map['is_max_level'] as bool? ?? false,
        checkedInToday: map['checked_in_today'] as bool? ?? false,
        consecutiveDays: (map['consecutive_days'] as num?)?.toInt(),
      );

  final int? level;
  final String? levelName;
  final String? color;
  final int? totalExp;
  final int? currentLevelExp;
  final int? nextLevel;
  final String? nextLevelName;
  final int? nextLevelExp;
  final int? remainingExp;
  final double? progress;
  final bool isMaxLevel;
  final bool checkedInToday;
  final int? consecutiveDays;
}

class CheckinStatusData {
  const CheckinStatusData({this.checkedInToday = false, this.consecutiveDays});

  factory CheckinStatusData.fromMap(Map<String, dynamic> map) =>
      CheckinStatusData(
        checkedInToday: map['checked_in_today'] as bool? ?? false,
        consecutiveDays: (map['consecutive_days'] as num?)?.toInt(),
      );

  final bool checkedInToday;
  final int? consecutiveDays;
}

class CheckinResultData {
  const CheckinResultData({
    this.expGained,
    this.consecutiveDays,
    this.level,
    this.levelName,
    this.totalExp,
  });

  factory CheckinResultData.fromMap(Map<String, dynamic> map) =>
      CheckinResultData(
        expGained: (map['exp_gained'] as num?)?.toInt(),
        consecutiveDays: (map['consecutive_days'] as num?)?.toInt(),
        level: (map['level'] as num?)?.toInt(),
        levelName: map['level_name'] as String?,
        totalExp: (map['total_exp'] as num?)?.toInt(),
      );

  final int? expGained;
  final int? consecutiveDays;
  final int? level;
  final String? levelName;
  final int? totalExp;
}

class ExperienceLogData {
  const ExperienceLogData({
    this.id,
    this.eventType,
    this.description,
    this.expDelta,
    this.createdAt,
  });

  factory ExperienceLogData.fromMap(Map<String, dynamic> map) =>
      ExperienceLogData(
        id: (map['id'] as num?)?.toInt(),
        eventType: map['event_type'] as String?,
        description: map['description'] as String?,
        expDelta: (map['exp_delta'] as num?)?.toInt(),
        createdAt: map['created_at'] as String?,
      );

  final int? id;
  final String? eventType;
  final String? description;
  final int? expDelta;
  final String? createdAt;
}

class PrivacySettingsData {
  const PrivacySettingsData({
    this.profileVisibility = 'public',
    this.showActivity = true,
    this.showBirthday = true,
    this.showComments = true,
    this.showFavorites = true,
    this.showLocation = true,
    this.showPosts = true,
    this.showRatings = true,
  });

  factory PrivacySettingsData.fromMap(Map<String, dynamic> map) =>
      PrivacySettingsData(
        profileVisibility: (map['profile_visibility'] as String?) ?? 'public',
        showActivity: map['show_activity'] as bool? ?? true,
        showBirthday: map['show_birthday'] as bool? ?? true,
        showComments: map['show_comments'] as bool? ?? true,
        showFavorites: map['show_favorites'] as bool? ?? true,
        showLocation: map['show_location'] as bool? ?? true,
        showPosts: map['show_posts'] as bool? ?? true,
        showRatings: map['show_ratings'] as bool? ?? true,
      );

  final String profileVisibility;
  final bool showActivity;
  final bool showBirthday;
  final bool showComments;
  final bool showFavorites;
  final bool showLocation;
  final bool showPosts;
  final bool showRatings;

  PrivacySettingsData copyWith({
    String? profileVisibility,
    bool? showActivity,
    bool? showBirthday,
    bool? showComments,
    bool? showFavorites,
    bool? showLocation,
    bool? showPosts,
    bool? showRatings,
  }) {
    return PrivacySettingsData(
      profileVisibility: profileVisibility ?? this.profileVisibility,
      showActivity: showActivity ?? this.showActivity,
      showBirthday: showBirthday ?? this.showBirthday,
      showComments: showComments ?? this.showComments,
      showFavorites: showFavorites ?? this.showFavorites,
      showLocation: showLocation ?? this.showLocation,
      showPosts: showPosts ?? this.showPosts,
      showRatings: showRatings ?? this.showRatings,
    );
  }
}

class ProfilePostData {
  const ProfilePostData({
    this.id,
    this.title,
    this.content,
    this.galgameId,
    this.galgameTitle,
    this.likeCount = 0,
    this.commentCount = 0,
    this.favoriteCount = 0,
    this.createdAt,
  });

  factory ProfilePostData.fromMap(Map<String, dynamic> map) =>
      ProfilePostData(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        content: map['content'] as String?,
        galgameId: (map['galgame_id'] as num?)?.toInt(),
        galgameTitle: map['galgame_title'] as String?,
        likeCount: (map['like_count'] as num?)?.toInt() ?? 0,
        commentCount: (map['comment_count'] as num?)?.toInt() ?? 0,
        favoriteCount: (map['favorite_count'] as num?)?.toInt() ?? 0,
        createdAt: map['created_at'] as String?,
      );

  final int? id;
  final String? title;
  final String? content;
  final int? galgameId;
  final String? galgameTitle;
  final int likeCount;
  final int commentCount;
  final int favoriteCount;
  final String? createdAt;
}

class ProfileCommentData {
  const ProfileCommentData({
    this.id,
    this.postId,
    this.postTitle,
    this.content,
    this.likeCount = 0,
    this.parentId,
    this.createdAt,
  });

  factory ProfileCommentData.fromMap(Map<String, dynamic> map) =>
      ProfileCommentData(
        id: (map['id'] as num?)?.toInt(),
        postId: (map['post_id'] as num?)?.toInt(),
        postTitle: map['post_title'] as String?,
        content: map['content'] as String?,
        likeCount: (map['like_count'] as num?)?.toInt() ?? 0,
        parentId: (map['parent_id'] as num?)?.toInt(),
        createdAt: map['created_at'] as String?,
      );

  final int? id;
  final int? postId;
  final String? postTitle;
  final String? content;
  final int likeCount;
  final int? parentId;
  final String? createdAt;
}

class ProfileGalgameData {
  const ProfileGalgameData({
    this.id,
    this.title,
    this.slug,
    this.coverUrl,
    this.coverSensitive = false,
    this.score,
    this.createdAt,
  });

  factory ProfileGalgameData.fromMap(Map<String, dynamic> map) =>
      ProfileGalgameData(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        slug: map['slug'] as String?,
        coverUrl: map['cover_url'] as String?,
        coverSensitive: map['cover_sensitive'] as bool? ?? false,
        score: (map['score'] as num?)?.toInt(),
        createdAt: map['created_at'] as String?,
      );

  final int? id;
  final String? title;
  final String? slug;
  final String? coverUrl;
  final bool coverSensitive;
  final int? score;
  final String? createdAt;
}

class UserActivityData {
  const UserActivityData({
    this.id,
    this.type,
    this.targetType,
    this.targetId,
    this.createdAt,
    this.metadata,
  });

  factory UserActivityData.fromMap(Map<String, dynamic> map) =>
      UserActivityData(
        id: (map['id'] as num?)?.toInt(),
        type: map['type'] as String?,
        targetType: map['target_type'] as String?,
        targetId: (map['target_id'] as num?)?.toInt(),
        createdAt: map['created_at'] as String?,
        metadata: map['metadata'] is Map
            ? Map<String, dynamic>.from(map['metadata'])
            : null,
      );

  final int? id;
  final String? type;
  final String? targetType;
  final int? targetId;
  final String? createdAt;
  final Map<String, dynamic>? metadata;
}
