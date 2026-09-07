class Paginated<T> {
  const Paginated({
    required this.items,
    required this.total,
    required this.page,
    required this.limit,
  });

  factory Paginated.fromMap(
    Map<String, dynamic> map, {
    T Function(Map<String, dynamic>)? fromItem,
    List<T> Function(List)? listBuilder,
  }) {
    final rawItems = (map['items'] as List?) ?? const [];
    final items = listBuilder != null
        ? listBuilder(rawItems)
        : rawItems
            .whereType<Map>()
            .map((e) => fromItem!(Map<String, dynamic>.from(e)))
            .toList();
    return Paginated<T>(
      items: items,
      total: (map['total'] as num?)?.toInt() ?? 0,
      page: (map['page'] as num?)?.toInt() ?? 1,
      limit: (map['limit'] as num?)?.toInt() ?? items.length,
    );
  }

  final List<T> items;
  final int total;
  final int page;
  final int limit;

  bool get hasMore => items.length + ((page - 1) * limit) < total;
}

