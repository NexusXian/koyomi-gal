enum RatingDimension {
  visual('visual', '画面'),
  story('story', '剧情'),
  music('music', '音乐'),
  character('character', '角色'),
  branch('branch', '分支'),
  system('system', '系统'),
  voice('voice', '配音'),
  replay('replay', '重玩');

  const RatingDimension(this.key, this.label);

  final String key;
  final String label;
}

enum RatingRecommendation {
  notRecommended(-1, '不推荐'),
  neutral(0, '中立'),
  recommended(1, '推荐'),
  stronglyRecommended(2, '强烈推荐');

  const RatingRecommendation(this.value, this.label);

  final int value;
  final String label;

  static RatingRecommendation? fromValue(int? value) {
    for (final recommendation in values) {
      if (recommendation.value == value) {
        return recommendation;
      }
    }
    return null;
  }
}

enum RatingSpoilerLevel {
  none(0, '无剧透'),
  mild(1, '部分剧透'),
  severe(2, '严重剧透');

  const RatingSpoilerLevel(this.value, this.label);

  final int value;
  final String label;

  static RatingSpoilerLevel? fromValue(int? value) {
    for (final level in values) {
      if (level.value == value) {
        return level;
      }
    }
    return null;
  }
}

enum RatingSort {
  newest('newest', '最新'),
  highest('highest', '最高分'),
  lowest('lowest', '最低分'),
  popular('popular', '最热门');

  const RatingSort(this.value, this.label);

  final String value;
  final String label;
}

class RatingDimensions {
  const RatingDimensions({
    this.visual,
    this.story,
    this.music,
    this.character,
    this.branch,
    this.system,
    this.voice,
    this.replay,
  });

  factory RatingDimensions.fromMap(Map<String, dynamic> map) =>
      RatingDimensions(
        visual: _intValue(map['visual'] ?? map['visual_score']),
        story: _intValue(map['story'] ?? map['story_score']),
        music: _intValue(map['music'] ?? map['music_score']),
        character: _intValue(map['character'] ?? map['character_score']),
        branch: _intValue(map['branch'] ?? map['branch_score']),
        system: _intValue(map['system'] ?? map['system_score']),
        voice: _intValue(map['voice'] ?? map['voice_score']),
        replay: _intValue(map['replay'] ?? map['replay_score']),
      );

  final int? visual;
  final int? story;
  final int? music;
  final int? character;
  final int? branch;
  final int? system;
  final int? voice;
  final int? replay;

  int? valueOf(RatingDimension dimension) => switch (dimension) {
    RatingDimension.visual => visual,
    RatingDimension.story => story,
    RatingDimension.music => music,
    RatingDimension.character => character,
    RatingDimension.branch => branch,
    RatingDimension.system => system,
    RatingDimension.voice => voice,
    RatingDimension.replay => replay,
  };

  bool get isEmpty =>
      RatingDimension.values.every((item) => valueOf(item) == null);

  bool get isPartial =>
      !isEmpty && RatingDimension.values.any((item) => valueOf(item) == null);

  Map<String, dynamic> toMap() => {
    'visual': visual,
    'story': story,
    'music': music,
    'character': character,
    'branch': branch,
    'system': system,
    'voice': voice,
    'replay': replay,
  };
}

class RatingDimensionSummary {
  const RatingDimensionSummary({this.average, this.count = 0});

  factory RatingDimensionSummary.fromMap(Map<String, dynamic> map) =>
      RatingDimensionSummary(
        average: _doubleValue(map['average'] ?? map['value']),
        count: _intValue(map['count']) ?? 0,
      );

  final double? average;
  final int count;
}

class GameRatingSummary {
  const GameRatingSummary({
    this.average,
    this.count = 0,
    this.dimensions = const {},
    this.recommendationCounts = const {},
  });

  factory GameRatingSummary.fromMap(Map<String, dynamic> map) {
    final overall = map['overall'] is Map
        ? Map<String, dynamic>.from(map['overall'] as Map)
        : const <String, dynamic>{};
    final rawDimensions = map['dimensions'] is Map
        ? Map<String, dynamic>.from(map['dimensions'] as Map)
        : const <String, dynamic>{};
    final dimensions = <RatingDimension, RatingDimensionSummary>{};

    for (final dimension in RatingDimension.values) {
      final raw = rawDimensions[dimension.key];
      if (raw is Map) {
        dimensions[dimension] = RatingDimensionSummary.fromMap(
          Map<String, dynamic>.from(raw),
        );
        continue;
      }
      final average = _doubleValue(
        map['${dimension.key}_average'] ??
            map['average_${dimension.key}'] ??
            map['${dimension.key}_score'],
      );
      final count = _intValue(map['${dimension.key}_count']) ?? 0;
      dimensions[dimension] = RatingDimensionSummary(
        average: average,
        count: count,
      );
    }

    final rawRecommendations =
        map['recommendation_counts'] ??
        map['recommendations'] ??
        map['recommendation_distribution'];
    final recommendations = <int, int>{};
    if (rawRecommendations is Map) {
      for (final entry in rawRecommendations.entries) {
        final key = int.tryParse(entry.key.toString());
        final value = _intValue(entry.value);
        if (key != null && value != null) {
          recommendations[key] = value;
        }
      }
    }

    return GameRatingSummary(
      average: _doubleValue(
        overall['average'] ??
            map['overall'] ??
            map['average'] ??
            map['average_score'],
      ),
      count: _intValue(overall['count'] ?? map['count'] ?? map['total']) ?? 0,
      dimensions: dimensions,
      recommendationCounts: recommendations,
    );
  }

  final double? average;
  final int count;
  final Map<RatingDimension, RatingDimensionSummary> dimensions;
  final Map<int, int> recommendationCounts;

  bool get isEmpty => count == 0 && average == null;

  RatingDimensionSummary dimension(RatingDimension dimension) =>
      dimensions[dimension] ?? const RatingDimensionSummary();
}

class GameRatingAuthor {
  const GameRatingAuthor({
    this.id,
    this.username,
    this.displayName,
    this.avatarUrl,
  });

  factory GameRatingAuthor.fromMap(Map<String, dynamic> map) =>
      GameRatingAuthor(
        id: _intValue(map['id']),
        username: map['username'] as String?,
        displayName: map['display_name'] as String?,
        avatarUrl: (map['avatar_url'] ?? map['avatar']) as String?,
      );

  final int? id;
  final String? username;
  final String? displayName;
  final String? avatarUrl;
}

class GameRating {
  const GameRating({
    this.id,
    this.galgameId,
    this.score,
    this.dimensions = const RatingDimensions(),
    this.recommendation,
    this.spoilerLevel = 0,
    this.review = '',
    this.editorMode = 'plain',
    this.playStatus,
    this.partial,
    this.likeCount = 0,
    this.liked = false,
    this.author,
    this.createdAt,
    this.updatedAt,
  });

  factory GameRating.fromMap(Map<String, dynamic> map) {
    final rawDimensions = map['dimensions'] is Map
        ? Map<String, dynamic>.from(map['dimensions'] as Map)
        : map;
    return GameRating(
      id: _intValue(map['id'] ?? map['rating_id']),
      galgameId: _intValue(map['galgame_id']),
      score: _intValue(map['score'] ?? map['overall'] ?? map['overall_score']),
      dimensions: RatingDimensions.fromMap(rawDimensions),
      recommendation: switch (map['recommendation']) {
        true => 1,
        false => -1,
        final value => _intValue(value),
      },
      spoilerLevel: _intValue(map['spoiler_level']) ?? 0,
      review:
          (map['review_text'] ?? map['review'] ?? map['content'] ?? '')
              as String,
      editorMode:
          (map['editor_mode'] ?? map['review_format'] ?? 'markdown') as String,
      playStatus: _intValue(map['play_status'] ?? map['state']),
      partial: map['partial'] as bool? ?? map['is_partial'] as bool?,
      likeCount: _intValue(map['like_count']) ?? 0,
      liked: map['liked'] as bool? ?? map['is_liked'] as bool? ?? false,
      author: map['author'] is Map
          ? GameRatingAuthor.fromMap(Map<String, dynamic>.from(map['author']))
          : map['user'] is Map
          ? GameRatingAuthor.fromMap(Map<String, dynamic>.from(map['user']))
          : null,
      createdAt: map['created_at'] as String?,
      updatedAt: map['updated_at'] as String?,
    );
  }

  final int? id;
  final int? galgameId;
  final int? score;
  final RatingDimensions dimensions;
  final int? recommendation;
  final int spoilerLevel;
  final String review;
  final String editorMode;
  final int? playStatus;
  final bool? partial;
  final int likeCount;
  final bool liked;
  final GameRatingAuthor? author;
  final String? createdAt;
  final String? updatedAt;

  bool get isPartial =>
      partial ??
      RatingDimension.values.any((item) => dimensions.valueOf(item) == null);

  GameRating copyWith({bool? liked, int? likeCount}) => GameRating(
    id: id,
    galgameId: galgameId,
    score: score,
    dimensions: dimensions,
    recommendation: recommendation,
    spoilerLevel: spoilerLevel,
    review: review,
    editorMode: editorMode,
    playStatus: playStatus,
    partial: partial,
    likeCount: likeCount ?? this.likeCount,
    liked: liked ?? this.liked,
    author: author,
    createdAt: createdAt,
    updatedAt: updatedAt,
  );
}

class GameRatingDraft {
  const GameRatingDraft({
    required this.score,
    this.dimensions = const RatingDimensions(),
    this.recommendation,
    this.spoilerLevel = 0,
    this.review = '',
    this.editorMode = 'plain',
  });

  factory GameRatingDraft.fromRating(GameRating rating) => GameRatingDraft(
    score: rating.score,
    dimensions: rating.dimensions,
    recommendation: rating.recommendation,
    spoilerLevel: rating.spoilerLevel,
    review: rating.review,
    editorMode: rating.editorMode,
  );

  final int? score;
  final RatingDimensions dimensions;
  final int? recommendation;
  final int spoilerLevel;
  final String review;
  final String editorMode;

  String? validate() {
    if (score == null) {
      return '请选择综合评分';
    }
    if (score! < 1 || score! > 10) {
      return '综合评分必须在 1 到 10 之间';
    }
    for (final dimension in RatingDimension.values) {
      final value = dimensions.valueOf(dimension);
      if (value != null && (value < 1 || value > 10)) {
        return '${dimension.label}评分必须在 1 到 10 之间';
      }
    }
    if (recommendation != null &&
        RatingRecommendation.fromValue(recommendation) == null) {
      return '推荐度选项无效';
    }
    if (RatingSpoilerLevel.fromValue(spoilerLevel) == null) {
      return '剧透等级选项无效';
    }
    if (editorMode != 'plain' && editorMode != 'markdown') {
      return '评价格式无效';
    }
    return null;
  }

  Map<String, dynamic> toMap() => {
    'overall': score,
    ...dimensions.toMap(),
    'recommendation': recommendation,
    'spoiler_level': spoilerLevel,
    'review_text': review.trim().isEmpty ? null : review.trim(),
  };
}

class GameRatingPage {
  const GameRatingPage({
    required this.items,
    required this.total,
    required this.page,
    required this.pageSize,
  });

  factory GameRatingPage.fromMap(Map<String, dynamic> map) {
    final rawItems = (map['items'] ?? map['list']) as List? ?? const [];
    final items = rawItems
        .whereType<Map>()
        .map((item) => GameRating.fromMap(Map<String, dynamic>.from(item)))
        .toList();
    return GameRatingPage(
      items: items,
      total: _intValue(map['total']) ?? items.length,
      page: _intValue(map['page']) ?? 1,
      pageSize: _intValue(map['page_size'] ?? map['limit']) ?? items.length,
    );
  }

  final List<GameRating> items;
  final int total;
  final int page;
  final int pageSize;

  bool get hasMore => page * pageSize < total;
}

class RatingLikeResult {
  const RatingLikeResult({required this.liked, required this.likeCount});

  factory RatingLikeResult.fromMap(Map<String, dynamic> map) =>
      RatingLikeResult(
        liked: map['liked'] as bool? ?? false,
        likeCount: _intValue(map['like_count']) ?? 0,
      );

  final bool liked;
  final int likeCount;
}

int? _intValue(Object? value) {
  if (value is num) {
    return value.toInt();
  }
  return int.tryParse(value?.toString() ?? '');
}

double? _doubleValue(Object? value) {
  if (value is num) {
    return value.toDouble();
  }
  return double.tryParse(value?.toString() ?? '');
}
