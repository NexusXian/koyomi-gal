class AdminPostData {
  const AdminPostData({
    this.id,
    this.authorId,
    this.authorName,
    this.galgameId,
    this.galgameTitle,
    this.title,
    this.content,
    this.editorMode,
    this.likeCount = 0,
    this.commentCount = 0,
    this.favoriteCount = 0,
    this.ip,
    this.ipRegion,
    this.createdAt,
    this.updatedAt,
  });

  factory AdminPostData.fromMap(Map<String, dynamic> map) => AdminPostData(
    id: (map['id'] as num?)?.toInt(),
    authorId: (map['author_id'] as num?)?.toInt(),
    authorName: map['author_name'] as String?,
    galgameId: (map['galgame_id'] as num?)?.toInt(),
    galgameTitle: map['galgame_title'] as String?,
    title: map['title'] as String?,
    content: map['content'] as String?,
    editorMode: map['editor_mode'] as String?,
    likeCount: (map['like_count'] as num?)?.toInt() ?? 0,
    commentCount: (map['comment_count'] as num?)?.toInt() ?? 0,
    favoriteCount: (map['favorite_count'] as num?)?.toInt() ?? 0,
    ip: map['ip'] as String?,
    ipRegion: map['ip_region'] as String?,
    createdAt: map['created_at'] as String?,
    updatedAt: map['updated_at'] as String?,
  );

  final int? id;
  final int? authorId;
  final String? authorName;
  final int? galgameId;
  final String? galgameTitle;
  final String? title;
  final String? content;
  final String? editorMode;
  final int likeCount;
  final int commentCount;
  final int favoriteCount;
  final String? ip;
  final String? ipRegion;
  final String? createdAt;
  final String? updatedAt;
}

class AdminCommentData {
  const AdminCommentData({
    this.id,
    this.postId,
    this.postTitle,
    this.authorId,
    this.authorName,
    this.parentId,
    this.replyToUserId,
    this.content,
    this.likeCount = 0,
    this.ip,
    this.ipRegion,
    this.createdAt,
    this.updatedAt,
  });

  factory AdminCommentData.fromMap(Map<String, dynamic> map) =>
      AdminCommentData(
        id: (map['id'] as num?)?.toInt(),
        postId: (map['post_id'] as num?)?.toInt(),
        postTitle: map['post_title'] as String?,
        authorId: (map['author_id'] as num?)?.toInt(),
        authorName: map['author_name'] as String?,
        parentId: (map['parent_id'] as num?)?.toInt(),
        replyToUserId: (map['reply_to_user_id'] as num?)?.toInt(),
        content: map['content'] as String?,
        likeCount: (map['like_count'] as num?)?.toInt() ?? 0,
        ip: map['ip'] as String?,
        ipRegion: map['ip_region'] as String?,
        createdAt: map['created_at'] as String?,
        updatedAt: map['updated_at'] as String?,
      );

  final int? id;
  final int? postId;
  final String? postTitle;
  final int? authorId;
  final String? authorName;
  final int? parentId;
  final int? replyToUserId;
  final String? content;
  final int likeCount;
  final String? ip;
  final String? ipRegion;
  final String? createdAt;
  final String? updatedAt;
}

class AdminUserRoleData {
  const AdminUserRoleData({this.id, this.name, this.code});

  factory AdminUserRoleData.fromMap(Map<String, dynamic> map) =>
      AdminUserRoleData(
        id: (map['id'] as num?)?.toInt(),
        name: map['name'] as String?,
        code: map['code'] as String?,
      );

  final int? id;
  final String? name;
  final String? code;
}

class AdminUserData {
  const AdminUserData({
    this.id,
    this.username,
    this.email,
    this.isBanned = false,
    this.roles = const [],
    this.createdAt,
    this.updatedAt,
  });

  factory AdminUserData.fromMap(Map<String, dynamic> map) => AdminUserData(
    id: (map['id'] as num?)?.toInt(),
    username: map['username'] as String?,
    email: map['email'] as String?,
    isBanned: map['is_banned'] as bool? ?? false,
    roles:
        (map['roles'] as List?)
            ?.whereType<Map>()
            .map(
              (item) =>
                  AdminUserRoleData.fromMap(Map<String, dynamic>.from(item)),
            )
            .toList() ??
        const [],
    createdAt: map['created_at'] as String?,
    updatedAt: map['updated_at'] as String?,
  );

  final int? id;
  final String? username;
  final String? email;
  final bool isBanned;
  final List<AdminUserRoleData> roles;
  final String? createdAt;
  final String? updatedAt;
}

class UserIPLog {
  const UserIPLog({
    this.id,
    this.userId,
    this.ip,
    this.country,
    this.region,
    this.city,
    this.isp,
    this.action,
    this.entityId,
    this.createdAt,
  });

  factory UserIPLog.fromMap(Map<String, dynamic> map) => UserIPLog(
    id: (map['id'] as num?)?.toInt(),
    userId: (map['user_id'] as num?)?.toInt(),
    ip: map['ip'] as String?,
    country: map['country'] as String?,
    region: map['region'] as String?,
    city: map['city'] as String?,
    isp: map['isp'] as String?,
    action: map['action'] as String?,
    entityId: (map['entity_id'] as num?)?.toInt(),
    createdAt: map['created_at'] as String?,
  );

  final int? id;
  final int? userId;
  final String? ip;
  final String? country;
  final String? region;
  final String? city;
  final String? isp;
  final String? action;
  final int? entityId;
  final String? createdAt;
}
