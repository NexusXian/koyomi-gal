import '../core/api/api_client.dart';
import '../models/novel_models.dart';
import '../models/pagination.dart';

class NovelService {
  NovelService(this._api);

  final ApiClient _api;

  Future<Paginated<NovelListItem>> list({
    String? keyword,
    List<int>? tagIds,
    String? author,
    String? publisher,
    String? label,
    String? releaseStatus,
    String? language,
    String? sort,
    int page = 1,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v1/novels',
      queryParameters: {
        'keyword': keyword,
        'tag_ids': tagIds == null || tagIds.isEmpty ? null : tagIds.join(','),
        'author': author,
        'publisher': publisher,
        'label': label,
        'release_status': releaseStatus,
        'language': language,
        'sort': sort,
        'page': page,
        'limit': limit,
      },
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: NovelListItem.fromMap,
    );
  }

  Future<NovelDetail> get(int id) async {
    final data = await _api.get('/api/v1/novels/$id');
    return NovelDetail.fromMap(Map<String, dynamic>.from(data));
  }

  Future<NovelDetail> create(Map<String, dynamic> payload) async {
    final data = await _api.post('/api/v1/novels', data: payload);
    return NovelDetail.fromMap(Map<String, dynamic>.from(data));
  }

  Future<NovelDetail> update(int id, Map<String, dynamic> payload) async {
    final data = await _api.put('/api/v1/novels/$id', data: payload);
    return NovelDetail.fromMap(Map<String, dynamic>.from(data));
  }

  Future<void> delete(int id) => _api.delete('/api/v1/novels/$id');

  Future<Paginated<VolumeData>> volumes(
    int id, {
    int page = 1,
    int limit = 100,
  }) async {
    final data = await _api.get(
      '/api/v1/novels/$id/volumes',
      queryParameters: {'page': page, 'limit': limit},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: VolumeData.fromMap,
    );
  }

  Future<VolumeData> getVolume(int id, int volumeId) async {
    final data = await _api.get('/api/v1/novels/$id/volumes/$volumeId');
    return VolumeData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<VolumeData> createVolume(int id, Map<String, dynamic> payload) async {
    final data = await _api.post('/api/v1/novels/$id/volumes', data: payload);
    return VolumeData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<VolumeData> updateVolume(
    int id,
    int volumeId,
    Map<String, dynamic> payload,
  ) async {
    final data = await _api.put(
      '/api/v1/novels/$id/volumes/$volumeId',
      data: payload,
    );
    return VolumeData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<void> deleteVolume(int id, int volumeId) =>
      _api.delete('/api/v1/novels/$id/volumes/$volumeId');

  Future<void> reorderVolumes(int id, List<int> volumeIds) =>
      _api.put('/api/v1/novels/$id/volumes/order', data: {'ids': volumeIds});
}
