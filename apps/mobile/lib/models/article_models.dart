class ArticleListItem {
  const ArticleListItem({
    this.id,
    this.title,
    this.summary,
    this.coverUrl,
    this.type,
    this.isPinned = false,
    this.viewCount = 0,
    this.publishedAt,
    this.createdAt,
    this.updatedAt,
  });

  factory ArticleListItem.fromMap(Map<String, dynamic> map) =>
      ArticleListItem(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        summary: map['summary'] as String?,
        coverUrl: map['cover_url'] as String?,
        type: map['type'] as String?,
        isPinned: map['is_pinned'] as bool? ?? false,
        viewCount: (map['view_count'] as num?)?.toInt() ?? 0,
        publishedAt: map['published_at'] as String?,
        createdAt: map['created_at'] as String?,
        updatedAt: map['updated_at'] as String?,
      );

  final int? id;
  final String? title;
  final String? summary;
  final String? coverUrl;
  final String? type;
  final bool isPinned;
  final int viewCount;
  final String? publishedAt;
  final String? createdAt;
  final String? updatedAt;
}

class ArticleDetail {
  const ArticleDetail({
    this.id,
    this.title,
    this.summary,
    this.content,
    this.coverUrl,
    this.type,
    this.editorMode = 'plain',
    this.isPinned = false,
    this.viewCount = 0,
    this.publishedAt,
    this.createdAt,
    this.updatedAt,
  });

  factory ArticleDetail.fromMap(Map<String, dynamic> map) => ArticleDetail(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        summary: map['summary'] as String?,
        content: map['content'] as String?,
        coverUrl: map['cover_url'] as String?,
        type: map['type'] as String?,
        editorMode: (map['editor_mode'] as String?) ?? 'plain',
        isPinned: map['is_pinned'] as bool? ?? false,
        viewCount: (map['view_count'] as num?)?.toInt() ?? 0,
        publishedAt: map['published_at'] as String?,
        createdAt: map['created_at'] as String?,
        updatedAt: map['updated_at'] as String?,
      );

  final int? id;
  final String? title;
  final String? summary;
  final String? content;
  final String? coverUrl;
  final String? type;
  final String editorMode;
  final bool isPinned;
  final int viewCount;
  final String? publishedAt;
  final String? createdAt;
  final String? updatedAt;
}
