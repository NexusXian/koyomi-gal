import 'package:dio/dio.dart';

import '../core/api/api_client.dart';
import '../core/config.dart';
import '../models/galgame_models.dart';
import '../models/pagination.dart';

class GalgameService {
  GalgameService(this._api);

  final ApiClient _api;

  Future<Paginated<GalgameListItem>> list({
    String? keyword,
    int? developerId,
    List<int>? tagIds,
    int? ageRating,
    String? sort,
    int page = 1,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v1/galgames',
      queryParameters: {
        'keyword': keyword,
        'developer_id': developerId,
        'tag_ids': tagIds == null || tagIds.isEmpty ? null : tagIds.join(','),
        'age_rating': ageRating,
        'sort': sort,
        'page': page,
        'limit': limit,
      },
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: GalgameListItem.fromMap,
    );
  }

  Future<GalgameDetail> get(int id) async {
    final data = await _api.get('/api/v1/galgames/$id');
    return GalgameDetail.fromMap(Map<String, dynamic>.from(data));
  }

  Future<GalgameDetail> create(Map<String, dynamic> payload) async {
    final data = await _api.post('/api/v1/galgames', data: payload);
    return GalgameDetail.fromMap(Map<String, dynamic>.from(data));
  }

  Future<GalgameDetail> update(int id, Map<String, dynamic> payload) async {
    final data = await _api.put('/api/v1/galgames/$id', data: payload);
    return GalgameDetail.fromMap(Map<String, dynamic>.from(data));
  }

  Future<List<GalgameCharacter>> characters(int id) async {
    final data = await _api.get('/api/v1/galgames/$id/characters');
    final items = ((data as Map?)?['items'] as List?) ?? const [];
    return items
        .whereType<Map>()
        .map((e) => GalgameCharacter.fromMap(Map<String, dynamic>.from(e)))
        .toList();
  }

  Future<List<GalleryImage>> gallery(int id) async {
    final data = await _api.get('/api/v1/galgames/$id/gallery');
    final items = ((data as Map?)?['items'] as List?) ?? const [];
    return items
        .whereType<Map>()
        .map((e) => GalleryImage.fromMap(Map<String, dynamic>.from(e)))
        .toList();
  }

  Future<Paginated<ContributorData>> contributors(
    int id, {
    int page = 1,
    int pageSize = 20,
  }) async {
    final data = await _api.get(
      '/api/v1/galgames/$id/contributors',
      queryParameters: {'page': page, 'page_size': pageSize},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: ContributorData.fromMap,
    );
  }

  Future<GalgameUserRelation> myRelation(int id) async {
    final data = await _api.get('/api/v1/galgames/$id/me');
    return GalgameUserRelation.fromMap(Map<String, dynamic>.from(data));
  }

  Future<void> addFavorite(int id) =>
      _api.post('/api/v1/galgames/$id/favorite');

  Future<void> removeFavorite(int id) =>
      _api.delete('/api/v1/galgames/$id/favorite');

  Future<void> upsertRating(int id, int score) =>
      _api.put('/api/v1/galgames/$id/rating', data: {'score': score});

  Future<void> deleteRating(int id) =>
      _api.delete('/api/v1/galgames/$id/rating');

  Future<void> upsertState(int id, int state, {int? playTimeMinutes}) =>
      _api.put(
        '/api/v1/galgames/$id/state',
        data: {'state': state, 'play_time_minutes': ?playTimeMinutes},
      );

  Future<void> deleteState(int id) => _api.delete('/api/v1/galgames/$id/state');

  Future<Paginated<GalgameListItem>> mine({
    String type = 'uploaded',
    int page = 1,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v1/me/galgames',
      queryParameters: {'type': type, 'page': page, 'limit': limit},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: GalgameListItem.fromMap,
    );
  }
}

class TagService {
  TagService(this._api);

  final ApiClient _api;

  Future<List<TagDataLite>> list() async {
    final data = await _api.get('/api/v1/tags');
    return (data as List)
        .whereType<Map>()
        .map((e) => TagDataLite.fromMap(Map<String, dynamic>.from(e)))
        .toList();
  }
}

class TagDataLite {
  const TagDataLite({this.id, this.name, this.slug});

  factory TagDataLite.fromMap(Map<String, dynamic> map) => TagDataLite(
        id: (map['id'] as num?)?.toInt(),
        name: map['name'] as String?,
        slug: map['slug'] as String?,
      );

  final int? id;
  final String? name;
  final String? slug;
}

class DeveloperService {
  DeveloperService(this._api);

  final ApiClient _api;

  Future<List<DeveloperDataLite>> list() async {
    final data = await _api.get('/api/v1/developers');
    return (data as List)
        .whereType<Map>()
        .map((e) => DeveloperDataLite.fromMap(Map<String, dynamic>.from(e)))
        .toList();
  }
}

class DeveloperDataLite {
  const DeveloperDataLite({this.id, this.name, this.originalName});

  factory DeveloperDataLite.fromMap(Map<String, dynamic> map) =>
      DeveloperDataLite(
        id: (map['id'] as num?)?.toInt(),
        name: map['name'] as String?,
        originalName: map['original_name'] as String?,
      );

  final int? id;
  final String? name;
  final String? originalName;
}

class ResourceService {
  ResourceService(this._api);

  final ApiClient _api;

  Future<Paginated<ResourceDataLite>> listByGalgame(
    int galgameId, {
    int page = 1,
    int limit = 20,
  }) => _list('/api/v2/galgames/$galgameId/resources', page, limit);

  Future<Paginated<ResourceDataLite>> listByNovel(
    int novelId, {
    int page = 1,
    int limit = 20,
  }) => _list('/api/v2/novels/$novelId/resources', page, limit);

  Future<Paginated<ResourceDataLite>> _list(
    String path,
    int page,
    int limit,
  ) async {
    final data = await _api.get(
      path,
      queryParameters: {'page': page, 'limit': limit},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: ResourceDataLite.fromMap,
    );
  }

  Future<ResourceDataLite> create(Map<String, dynamic> payload) async {
    final data = await _api.post('/api/v1/resources', data: payload);
    return ResourceDataLite.fromMap(Map<String, dynamic>.from(data));
  }

  Future<ResourceDataLite> update(int id, Map<String, dynamic> payload) async {
    final data = await _api.put('/api/v1/resources/$id', data: payload);
    return ResourceDataLite.fromMap(Map<String, dynamic>.from(data));
  }

  Future<void> delete(int id) => _api.delete('/api/v1/resources/$id');

  Future<void> report(
    int id, {
    required int reason,
    required String description,
  }) => _api.post(
    '/api/v1/resources/$id/reports',
    data: {'reason': reason, 'description': description},
  );
}

class ResourceDataLite {
  const ResourceDataLite({
    this.id,
    this.title,
    this.description,
    this.type,
    this.status,
    this.targetType,
    this.targetId,
    this.uploaderId,
    this.uploader,
    this.createdAt,
    this.updatedAt,
    this.links = const [],
  });

  factory ResourceDataLite.fromMap(Map<String, dynamic> map) =>
      ResourceDataLite(
        id: (map['id'] as num?)?.toInt(),
        title: map['title'] as String?,
        description: map['description'] as String?,
        type: (map['type'] as num?)?.toInt(),
        status: (map['status'] as num?)?.toInt(),
        targetType: map['target_type'] as String?,
        targetId: (map['target_id'] as num?)?.toInt(),
        uploaderId: (map['uploader_id'] as num?)?.toInt(),
        uploader: map['uploader'] is Map
            ? ResourceUploaderLite.fromMap(
                Map<String, dynamic>.from(map['uploader']),
              )
            : null,
        createdAt: map['created_at'] as String?,
        updatedAt: map['updated_at'] as String?,
        links:
            (map['links'] as List?)
                ?.whereType<Map>()
                .map(
                  (e) => ResourceLinkLite.fromMap(Map<String, dynamic>.from(e)),
                )
                .toList() ??
            const [],
      );

  final int? id;
  final String? title;
  final String? description;
  final int? type;
  final int? status;
  final String? targetType;
  final int? targetId;
  final int? uploaderId;
  final ResourceUploaderLite? uploader;
  final String? createdAt;
  final String? updatedAt;
  final List<ResourceLinkLite> links;
}

class ResourceUploaderLite {
  const ResourceUploaderLite({
    this.id,
    this.username,
    this.displayName,
    this.avatarUrl,
  });

  factory ResourceUploaderLite.fromMap(Map<String, dynamic> map) =>
      ResourceUploaderLite(
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

class ResourceLinkLite {
  const ResourceLinkLite({this.id, this.url});

  factory ResourceLinkLite.fromMap(Map<String, dynamic> map) =>
      ResourceLinkLite(
        id: (map['id'] as num?)?.toInt(),
        url: map['url'] as String?,
      );

  final int? id;
  final String? url;
}

class ImageService {
  ImageService(this._api, this._dio);

  final ApiClient _api;
  final Dio _dio;

  Future<({int id, String uploadUrl})> presign({
    required String filename,
    required String contentType,
    required int size,
    required String category,
  }) async {
    final data = await _api.post(
      '/api/v1/images/presign',
      data: {
        'filename': filename,
        'content_type': contentType,
        'size': size,
        'category': category,
      },
    );
    final map = Map<String, dynamic>.from(data);
    return (
      id: (map['id'] as num?)?.toInt() ?? 0,
      uploadUrl: (map['upload_url'] as String?) ?? '',
    );
  }

  Future<void> uploadToPresigned(
    String url,
    List<int> bytes,
    String mime,
  ) async {
    await _dio.put(
      url,
      data: bytes,
      options: Options(
        headers: {
          Headers.contentLengthHeader: bytes.length,
          'Content-Type': mime,
        },
      ),
    );
  }

  Future<Map<String, dynamic>> complete(
    int id, {
    int? width,
    int? height,
  }) async {
    final data = await _api.post(
      '/api/v1/images/$id/complete',
      data: {'width': ?width, 'height': ?height},
    );
    return Map<String, dynamic>.from(data);
  }
}

const apiPrefix = AppConfig.apiPrefix;
