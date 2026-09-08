class Announcement {
  const Announcement({
    required this.id,
    required this.title,
    required this.content,
    required this.type,
    required this.displayMode,
    required this.target,
    required this.priority,
    required this.dismissible,
    this.startsAt,
    this.endsAt,
    this.updatedAt,
  });

  factory Announcement.fromMap(Map<String, dynamic> map) {
    return Announcement(
      id: map['id']?.toString() ?? '',
      title: map['title'] as String? ?? '',
      content: map['content'] as String? ?? '',
      type: map['type'] as String? ?? '',
      displayMode: map['displayMode'] as String? ?? '',
      target: map['target'] as String? ?? '',
      priority: (map['priority'] as num?)?.toInt() ?? 0,
      dismissible: map['dismissible'] as bool? ?? true,
      startsAt: DateTime.tryParse(map['startsAt'] as String? ?? ''),
      endsAt: DateTime.tryParse(map['endsAt'] as String? ?? ''),
      updatedAt: DateTime.tryParse(map['updatedAt'] as String? ?? ''),
    );
  }

  final String id;
  final String title;
  final String content;
  final String type;
  final String displayMode;
  final String target;
  final int priority;
  final bool dismissible;
  final DateTime? startsAt;
  final DateTime? endsAt;
  final DateTime? updatedAt;

  bool get isImportant {
    const importantTypes = {'warning', 'maintenance', 'system'};
    return !dismissible || importantTypes.contains(type.trim().toLowerCase());
  }

  bool targetsPlatform(String platform) {
    final normalized = target.trim().toLowerCase();
    return normalized.isEmpty || normalized == 'all' || normalized == platform;
  }
}
