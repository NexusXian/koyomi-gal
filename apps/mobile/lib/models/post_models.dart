import 'user_models.dart';

class PostData {
  const PostData({
    this.id,
    this.title,
    this.content,
    this.editorMode = 'plain',
    this.author,
    this.authorId,
    this.authorName,
    this.authorAvatar,
    this.galgameId,
    this.galgameTitle,
    this.likeCount = 0,
    this.favoriteCount = 0,
    this.commentCount = 0,
    this.createdAt,
    this.updatedAt,
  });

  factory PostData.fromMap(Map<String, dynamic> map) => PostData(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        content: map['content'] as String?,
        editorMode: (map['editor_mode'] as String?) ?? 'plain',
        author: map['author'] is Map
            ? CommunityUserSummary.fromMap(
                Map<String, dynamic>.from(map['author']))
            : null,
        authorId: (map['author_id'] as num?)?.toInt(),
        authorName: map['author_name'] as String?,
        authorAvatar: map['author_avatar'] as String?,
        galgameId: (map['galgame_id'] as num?)?.toInt(),
        galgameTitle: map['galgame_title'] as String?,
        likeCount: (map['like_count'] as num?)?.toInt() ?? 0,
        favoriteCount: (map['favorite_count'] as num?)?.toInt() ?? 0,
        commentCount: (map['comment_count'] as num?)?.toInt() ?? 0,
        createdAt: map['created_at'] as String?,
        updatedAt: map['updated_at'] as String?,
      );

  final int? id;
  final String? title;
  final String? content;
  final String editorMode;
  final CommunityUserSummary? author;
  final int? authorId;
  final String? authorName;
  final String? authorAvatar;
  final int? galgameId;
  final String? galgameTitle;
  final int likeCount;
  final int favoriteCount;
  final int commentCount;
  final String? createdAt;
  final String? updatedAt;
}

class PostLikeData {
  const PostLikeData({this.liked = false, this.likeCount = 0});

  factory PostLikeData.fromMap(Map<String, dynamic> map) => PostLikeData(
        liked: map['liked'] as bool? ?? false,
        likeCount: (map['like_count'] as num?)?.toInt() ?? 0,
      );

  final bool liked;
  final int likeCount;
}

class PostFavoriteData {
  const PostFavoriteData({this.favorited = false, this.favoriteCount = 0});

  factory PostFavoriteData.fromMap(Map<String, dynamic> map) =>
      PostFavoriteData(
        favorited: map['favorited'] as bool? ?? false,
        favoriteCount: (map['favorite_count'] as num?)?.toInt() ?? 0,
      );

  final bool favorited;
  final int favoriteCount;
}

class HomePost {
  const HomePost({
    this.id,
    this.title,
    this.author,
    this.galgame,
    this.likeCount = 0,
    this.commentCount = 0,
    this.favoriteCount = 0,
    this.createdAt,
  });

  factory HomePost.fromMap(Map<String, dynamic> map) => HomePost(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        author: map['author'] is Map
            ? CommunityUserSummary.fromMap(
                Map<String, dynamic>.from(map['author']))
            : null,
        galgame: map['galgame'] is Map
            ? PostGalgame.fromMap(Map<String, dynamic>.from(map['galgame']))
            : null,
        likeCount: (map['like_count'] as num?)?.toInt() ?? 0,
        commentCount: (map['comment_count'] as num?)?.toInt() ?? 0,
        favoriteCount: (map['favorite_count'] as num?)?.toInt() ?? 0,
        createdAt: map['created_at'] as String?,
      );

  final int? id;
  final String? title;
  final CommunityUserSummary? author;
  final PostGalgame? galgame;
  final int likeCount;
  final int commentCount;
  final int favoriteCount;
  final String? createdAt;
}

class PostGalgame {
  const PostGalgame({this.id, this.title, this.coverUrl});

  factory PostGalgame.fromMap(Map<String, dynamic> map) => PostGalgame(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        coverUrl: map['cover_url'] as String?,
      );

  final int? id;
  final String? title;
  final String? coverUrl;
}

class CommentData {
  const CommentData({
    this.id,
    this.postId,
    this.parentId,
    this.author,
    this.replyTo,
    this.content,
    this.likeCount = 0,
    this.replyCount = 0,
    this.createdAt,
  });

  factory CommentData.fromMap(Map<String, dynamic> map) => CommentData(
        id: (map['id'] as num?)?.toInt(),
        postId: (map['post_id'] as num?)?.toInt(),
        parentId: (map['parent_id'] as num?)?.toInt(),
        author: map['author'] is Map
            ? CommunityUserSummary.fromMap(
                Map<String, dynamic>.from(map['author']))
            : null,
        replyTo: map['reply_to'] is Map
            ? CommunityUserSummary.fromMap(
                Map<String, dynamic>.from(map['reply_to']))
            : null,
        content: map['content'] as String?,
        likeCount: (map['like_count'] as num?)?.toInt() ?? 0,
        replyCount: (map['reply_count'] as num?)?.toInt() ?? 0,
        createdAt: map['created_at'] as String?,
      );

  final int? id;
  final int? postId;
  final int? parentId;
  final CommunityUserSummary? author;
  final CommunityUserSummary? replyTo;
  final String? content;
  final int likeCount;
  final int replyCount;
  final String? createdAt;
}

class CommentLikeData {
  const CommentLikeData({this.liked = false, this.likeCount = 0});

  factory CommentLikeData.fromMap(Map<String, dynamic> map) =>
      CommentLikeData(
        liked: map['liked'] as bool? ?? false,
        likeCount: (map['like_count'] as num?)?.toInt() ?? 0,
      );

  final bool liked;
  final int likeCount;
}
