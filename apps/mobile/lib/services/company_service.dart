import '../core/api/api_client.dart';
import '../models/company_models.dart';
import '../models/galgame_models.dart';
import '../models/pagination.dart';

class CompanyService {
  CompanyService(this._api);

  final ApiClient _api;

  Future<Paginated<Company>> list({
    bool admin = false,
    String? keyword,
    String? status,
    String? country,
    String sort = 'created_at',
    int page = 1,
    int pageSize = 20,
  }) async {
    final data = await _api.get(
      admin ? '/api/v1/admin/companies' : '/api/v1/companies',
      queryParameters: {
        'keyword': keyword,
        'status': status,
        'country': country,
        'sort': sort,
        'page': page,
        'page_size': pageSize,
      },
    );
    final map = Map<String, dynamic>.from(data);
    return Paginated.fromMap({
      ...map,
      'limit': map['page_size'] ?? pageSize,
    }, fromItem: Company.fromMap);
  }

  Future<Company> get(String slug) async => Company.fromMap(
    Map<String, dynamic>.from(
      await _api.get('/api/v1/companies/${Uri.encodeComponent(slug)}'),
    ),
  );

  Future<Company> adminGet(int id) async => Company.fromMap(
    Map<String, dynamic>.from(await _api.get('/api/v1/admin/companies/$id')),
  );

  Future<Company> create(CompanyRequest request) async => Company.fromMap(
    Map<String, dynamic>.from(
      await _api.post('/api/v1/admin/companies', data: request.toMap()),
    ),
  );

  Future<Company> update(int id, CompanyRequest request) async =>
      Company.fromMap(
        Map<String, dynamic>.from(
          await _api.put('/api/v1/admin/companies/$id', data: request.toMap()),
        ),
      );

  Future<void> delete(int id) => _api.delete('/api/v1/admin/companies/$id');

  Future<Paginated<GalgameListItem>> games(
    String slug, {
    int page = 1,
    int pageSize = 20,
    String? role,
    String? keyword,
    String? sort,
  }) async => Paginated.fromMap(
    Map<String, dynamic>.from(
      await _api.get(
        '/api/v1/companies/${Uri.encodeComponent(slug)}/games',
        queryParameters: {
          'page': page,
          'page_size': pageSize,
          'role': role,
          'keyword': keyword,
          'sort': sort,
        },
      ),
    ),
    fromItem: GalgameListItem.fromMap,
  );

  Future<void> replaceGameCompanies(int id, List<GameCompany> companies) =>
      _api.put(
        '/api/v1/admin/games/$id/companies',
        data: {'companies': companies.map((e) => e.toBinding()).toList()},
      );
}
