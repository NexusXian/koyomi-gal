import '../core/utils/company_slug.dart';

const companyStatuses = {'active': '运营中', 'inactive': '暂停运营', 'closed': '已关闭'};

const companyRoles = {
  'developer': '开发商',
  'publisher': '发行商',
  'brand': '品牌',
  'co_developer': '共同开发',
};

class CompanySummary {
  const CompanySummary({
    required this.id,
    required this.name,
    required this.slug,
    this.logoUrl,
  });

  factory CompanySummary.fromMap(Map<String, dynamic> map) => CompanySummary(
    id: (map['id'] as num).toInt(),
    name: map['name'] as String? ?? '',
    slug: map['slug'] as String? ?? '',
    logoUrl: map['logo_url'] as String?,
  );

  final int id;
  final String name;
  final String slug;
  final String? logoUrl;
}

class Company extends CompanySummary {
  const Company({
    required super.id,
    required super.name,
    required super.slug,
    super.logoUrl,
    this.nameJa,
    this.nameZh,
    this.nameEn,
    this.description,
    this.bannerUrl,
    this.websiteUrl,
    this.country,
    this.foundedDate,
    this.closedDate,
    this.status = 'active',
    this.parentCompanyId,
    this.parentCompany,
    this.children = const [],
    this.aliases = const [],
    this.gameCount = 0,
    this.developerGameCount = 0,
    this.publisherGameCount = 0,
    this.createdAt,
    this.updatedAt,
  });

  factory Company.fromMap(Map<String, dynamic> map) => Company(
    id: (map['id'] as num).toInt(),
    name: map['name'] as String? ?? '',
    slug: map['slug'] as String? ?? '',
    logoUrl: map['logo_url'] as String?,
    nameJa: map['name_ja'] as String?,
    nameZh: map['name_zh'] as String?,
    nameEn: map['name_en'] as String?,
    description: map['description'] as String?,
    bannerUrl: map['banner_url'] as String?,
    websiteUrl: map['website_url'] as String?,
    country: map['country'] as String?,
    foundedDate: map['founded_date'] as String?,
    closedDate: map['closed_date'] as String?,
    status: map['status'] as String? ?? 'active',
    parentCompanyId: (map['parent_company_id'] as num?)?.toInt(),
    parentCompany: map['parent_company'] is Map
        ? CompanySummary.fromMap(
            Map<String, dynamic>.from(map['parent_company']),
          )
        : null,
    children: (map['children'] as List? ?? const [])
        .whereType<Map>()
        .map((e) => CompanySummary.fromMap(Map<String, dynamic>.from(e)))
        .toList(),
    aliases: (map['aliases'] as List? ?? const []).whereType<String>().toList(),
    gameCount: (map['game_count'] as num?)?.toInt() ?? 0,
    developerGameCount: (map['developer_game_count'] as num?)?.toInt() ?? 0,
    publisherGameCount: (map['publisher_game_count'] as num?)?.toInt() ?? 0,
    createdAt: map['created_at'] as String?,
    updatedAt: map['updated_at'] as String?,
  );

  final String? nameJa, nameZh, nameEn, description, bannerUrl, websiteUrl;
  final String? country, foundedDate, closedDate, createdAt, updatedAt;
  final String status;
  final int? parentCompanyId;
  final CompanySummary? parentCompany;
  final List<CompanySummary> children;
  final List<String> aliases;
  final int gameCount, developerGameCount, publisherGameCount;
}

class GameCompany extends CompanySummary {
  const GameCompany({
    required super.id,
    required super.name,
    required super.slug,
    super.logoUrl,
    required this.role,
  });

  factory GameCompany.fromMap(Map<String, dynamic> map) => GameCompany(
    id: (map['id'] as num).toInt(),
    name: map['name'] as String? ?? '',
    slug: map['slug'] as String? ?? '',
    logoUrl: map['logo_url'] as String?,
    role: map['role'] as String? ?? 'developer',
  );

  final String role;

  Map<String, dynamic> toBinding() => {'company_id': id, 'role': role};
}

class CompanyRequest {
  const CompanyRequest({
    required this.name,
    required this.slug,
    this.nameJa = '',
    this.nameZh = '',
    this.nameEn = '',
    this.description = '',
    this.logoUrl = '',
    this.bannerUrl = '',
    this.websiteUrl = '',
    this.country = '',
    this.foundedDate,
    this.closedDate,
    this.status = 'active',
    this.parentCompanyId,
    this.aliases = const [],
  });

  final String name, slug, nameJa, nameZh, nameEn, description;
  final String logoUrl, bannerUrl, websiteUrl, country, status;
  final String? foundedDate, closedDate;
  final int? parentCompanyId;
  final List<String> aliases;

  Map<String, dynamic> toMap() => {
    'name': name,
    'slug': normalizeCompanySlug(slug),
    'name_ja': nameJa,
    'name_zh': nameZh,
    'name_en': nameEn,
    'description': description,
    'logo_url': logoUrl,
    'banner_url': bannerUrl,
    'website_url': websiteUrl,
    'country': country,
    'founded_date': foundedDate,
    'closed_date': closedDate,
    'status': status,
    'parent_company_id': parentCompanyId,
    'aliases': aliases,
  };
}
