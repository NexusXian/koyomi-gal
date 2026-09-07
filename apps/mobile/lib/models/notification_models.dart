class ActorData {
  const ActorData({
    this.id,
    this.username,
    this.displayName,
    this.avatarUrl,
  });

  factory ActorData.fromMap(Map<String, dynamic> map) => ActorData(
        id: (map['id'] as num?)?.toInt(),
        username: map['username'] as String?,
        displayName: map['display_name'] as String?,
        avatarUrl: map['avatar_url'] as String?,
      );

  final int? id;
  final String? username;
  final String? displayName;
  final String? avatarUrl;
}

class NotificationData {
  const NotificationData({
    this.id,
    this.title,
    this.content,
    this.type,
    this.category,
    this.entityType,
    this.entityId,
    this.targetUrl,
    this.isRead = false,
    this.readAt,
    this.actor,
    this.createdAt,
  });

  factory NotificationData.fromMap(Map<String, dynamic> map) =>
      NotificationData(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        content: map['content'] as String?,
        type: map['type'] as String?,
        category: map['category'] as String?,
        entityType: map['entity_type'] as String?,
        entityId: (map['entity_id'] as num?)?.toInt(),
        targetUrl: map['target_url'] as String?,
        isRead: map['is_read'] as bool? ?? false,
        readAt: map['read_at'] as String?,
        actor: map['actor'] is Map
            ? ActorData.fromMap(Map<String, dynamic>.from(map['actor']))
            : null,
        createdAt: map['created_at'] as String?,
      );

  final int? id;
  final String? title;
  final String? content;
  final String? type;
  final String? category;
  final String? entityType;
  final int? entityId;
  final String? targetUrl;
  final bool isRead;
  final String? readAt;
  final ActorData? actor;
  final String? createdAt;
}
