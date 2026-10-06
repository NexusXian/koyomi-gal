import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/core/utils/company_slug.dart';
import 'package:mobile/models/company_models.dart';
import 'package:mobile/models/galgame_models.dart';

void main() {
  test('company slug normalization preserves Unicode and internal spaces', () {
    final payload = const CompanyRequest(
      name: 'Brand',
      slug: '  製作会社 BRÄND 名  ',
    ).toMap();
    expect(payload['slug'], '製作会社 bränd 名');
    expect(normalizeCompanySlug('  BRAND\u3000'), 'brand');
    expect(normalizeCompanySlug(' \uFEFFBRAND\uFEFF '), '\uFEFFbrand\uFEFF');
  });

  test('company slug validation allows backfilled Unicode and punctuation', () {
    for (final slug in [
      '日本 語',
      '会社🌸',
      'BRÄND Name',
      'brand.name',
      '...name',
      '品牌／名称',
      'brand\u200Dname',
      'brand%20name',
      '\uFEFFbrand',
    ]) {
      expect(isValidCompanySlug(slug), isTrue, reason: slug);
    }
  });

  test('company slug validation rejects empty and path-component slugs', () {
    for (final slug in ['', '  ', '.', ' .. ', 'brand/name', r'brand\name']) {
      expect(isValidCompanySlug(slug), isFalse, reason: slug);
    }
  });

  test('company slug validation rejects every Cc control character', () {
    for (final code in [
      ...List.generate(32, (index) => index),
      ...List.generate(33, (index) => 0x7f + index),
    ]) {
      expect(
        isValidCompanySlug('brand${String.fromCharCode(code)}name'),
        isFalse,
        reason: 'U+${code.toRadixString(16)}',
      );
    }
  });

  test('company detail preserves nullable dates, hierarchy and counts', () {
    final company = Company.fromMap({
      'id': 12,
      'name': 'Example Brand',
      'slug': 'example-brand',
      'name_ja': 'ブランド',
      'name_zh': '品牌',
      'name_en': 'Example Brand',
      'logo_url': 'https://example.test/logo.png',
      'banner_url': 'https://example.test/banner.png',
      'website_url': 'https://example.test',
      'description': '**Brand**',
      'country': 'JP',
      'founded_date': '2001-02-03',
      'closed_date': null,
      'status': 'active',
      'aliases': ['Brand', 'ブランド'],
      'parent_company_id': 2,
      'parent_company': {
        'id': 2,
        'name': 'Parent',
        'slug': 'parent',
        'logo_url': null,
      },
      'children': [
        {'id': 13, 'name': 'Child', 'slug': 'child'},
      ],
      'game_count': 8,
      'developer_game_count': 5,
      'publisher_game_count': 4,
      'created_at': '2026-01-01T00:00:00Z',
      'updated_at': '2026-10-05T09:30:00Z',
    });

    expect(company.foundedDate, '2001-02-03');
    expect(company.closedDate, isNull);
    expect(company.nameJa, 'ブランド');
    expect(company.nameZh, '品牌');
    expect(company.nameEn, 'Example Brand');
    expect(company.logoUrl, 'https://example.test/logo.png');
    expect(company.bannerUrl, 'https://example.test/banner.png');
    expect(company.websiteUrl, 'https://example.test');
    expect(company.description, '**Brand**');
    expect(company.country, 'JP');
    expect(company.aliases, ['Brand', 'ブランド']);
    expect(company.parentCompanyId, company.parentCompany!.id);
    expect(company.parentCompany!.slug, 'parent');
    expect(company.children.single.slug, 'child');
    expect(
      [
        company.gameCount,
        company.developerGameCount,
        company.publisherGameCount,
      ],
      [8, 5, 4],
    );
    expect(company.createdAt, '2026-01-01T00:00:00Z');
    expect(company.updatedAt, '2026-10-05T09:30:00Z');
  });

  test('null company collections and parent are empty rather than errors', () {
    final company = Company.fromMap({
      'id': 12,
      'aliases': null,
      'children': null,
      'parent_company': null,
    });
    expect(company.aliases, isEmpty);
    expect(company.children, isEmpty);
    expect(company.parentCompany, isNull);
    expect(company.foundedDate, isNull);
    expect(company.gameCount, 0);
  });

  test(
    'company update explicitly clears nullable dates, parent and aliases',
    () {
      final payload = const CompanyRequest(
        name: 'Brand',
        slug: 'brand',
        status: 'inactive',
      ).toMap();
      expect(payload, {
        'name': 'Brand',
        'slug': 'brand',
        'name_ja': '',
        'name_zh': '',
        'name_en': '',
        'description': '',
        'logo_url': '',
        'banner_url': '',
        'website_url': '',
        'country': '',
        'founded_date': null,
        'closed_date': null,
        'status': 'inactive',
        'parent_company_id': null,
        'aliases': <String>[],
      });
    },
  );

  test('galgame company array keeps multiple roles and binding IDs', () {
    final detail = GalgameDetail.fromMap({
      'id': 41,
      'companies': [
        {
          'id': 12,
          'name': 'Brand',
          'slug': 'brand',
          'logo_url': null,
          'role': 'developer',
        },
        {
          'id': 12,
          'name': 'Brand',
          'slug': 'brand',
          'logo_url': null,
          'role': 'publisher',
        },
      ],
    });
    expect(detail.companies.map((company) => company.toBinding()).toList(), [
      {'company_id': 12, 'role': 'developer'},
      {'company_id': 12, 'role': 'publisher'},
    ]);
    expect(GalgameDetail.fromMap({'companies': null}).companies, isEmpty);
  });
}
