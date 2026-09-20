import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/core/utils/game_descriptions.dart';
import 'package:mobile/models/galgame_models.dart';
import 'package:mobile/widgets/markdown_view.dart';

Map<String, GameDescription> descriptionsFrom(Map<String, dynamic> json) {
  return (json['descriptions'] as Map<String, dynamic>? ?? {}).map(
    (key, value) => MapEntry(
      key,
      GameDescription.fromMap(Map<String, dynamic>.from(value)),
    ),
  );
}

void main() {
  group('GalgameDetail descriptions parsing', () {
    test('parses the language-keyed descriptions map', () {
      final detail = GalgameDetail.fromMap({
        'id': 1,
        'title': 'Summer Pockets',
        'descriptions': {
          'zh-CN': {
            'language': 'zh-CN',
            'content': '中文简介',
            'source': {
              'type': 'nextmoe',
              'name': 'NextMoe 资料库',
              'official': false,
            },
          },
          'en-US': {
            'language': 'en-US',
            'content': 'English description',
            'source': {
              'type': 'vndb',
              'name': 'VNDB',
              'url': 'https://vndb.org/v20424',
              'official': false,
            },
          },
          'ja-JP': {
            'language': 'ja-JP',
            'content': '日本語紹介',
            'source': {
              'type': 'official',
              'name': '游戏官网',
              'url': 'https://example.jp',
              'official': true,
            },
          },
        },
      });

      expect(detail.descriptions.length, 3);
      final zh = detail.descriptions['zh-CN']!;
      expect(zh.content, '中文简介');
      expect(zh.source.type, 'nextmoe');
      expect(zh.source.name, 'NextMoe 资料库');
      expect(zh.source.url, isNull);
      expect(zh.source.official, isFalse);

      final en = detail.descriptions['en-US']!;
      expect(en.source.url, 'https://vndb.org/v20424');
      expect(en.source.official, isFalse);

      final ja = detail.descriptions['ja-JP']!;
      expect(ja.source.official, isTrue);
    });

    test('missing descriptions default to an empty map', () {
      final detail = GalgameDetail.fromMap({'id': 2, 'title': 'Game'});
      expect(detail.descriptions, isEmpty);
    });

    test('missing source falls back to unknown defaults', () {
      final description = GameDescription.fromMap({
        'language': 'zh-CN',
        'content': 'x',
      });
      expect(description.source.type, 'unknown');
      expect(description.source.official, isFalse);
    });
  });

  group('normalizeLocale', () {
    test('maps language codes onto stored locales', () {
      expect(normalizeLocale(const Locale('zh')), 'zh-CN');
      expect(normalizeLocale(const Locale('zh', 'CN')), 'zh-CN');
      expect(normalizeLocale(const Locale('ja')), 'ja-JP');
      expect(normalizeLocale(const Locale('en')), 'en-US');
      expect(normalizeLocale(const Locale('fr')), 'en-US');
    });
  });

  group('getBestDescription', () {
    test('returns the locale description when present', () {
      final best = getBestDescription(
        descriptionsFrom({
          'descriptions': {
            'zh-CN': {
              'language': 'zh-CN',
              'content': '中文',
              'source': {'type': 'nextmoe', 'name': 'NextMoe 资料库'},
            },
            'en-US': {
              'language': 'en-US',
              'content': 'English',
              'source': {'type': 'vndb', 'name': 'VNDB'},
            },
          },
        }),
        'zh-CN',
      );

      expect(best, isNotNull);
      expect(best!.description.content, '中文');
      expect(best.isFallback, isFalse);
    });

    test('falls back zh-CN -> ja-JP -> en-US', () {
      final descriptions = descriptionsFrom({
        'descriptions': {
          'en-US': {
            'language': 'en-US',
            'content': 'English',
            'source': {'type': 'vndb', 'name': 'VNDB'},
          },
        },
      });

      final best = getBestDescription(descriptions, 'zh-CN');
      expect(best!.description.content, 'English');
      expect(best.locale, 'en-US');
      expect(best.isFallback, isTrue);

      final jaFirst = getBestDescription(descriptions, 'ja-JP');
      expect(jaFirst!.description.content, 'English');
      expect(jaFirst.locale, 'en-US');
    });

    test('falls back ja-JP to zh-CN before en-US', () {
      final best = getBestDescription(
        descriptionsFrom({
          'descriptions': {
            'zh-CN': {
              'language': 'zh-CN',
              'content': '中文',
              'source': {'type': 'nextmoe', 'name': 'NextMoe 资料库'},
            },
          },
        }),
        'ja-JP',
      );

      expect(best!.locale, 'zh-CN');
      expect(best.isFallback, isTrue);
    });

    test('skips languages without usable content', () {
      final best = getBestDescription(
        descriptionsFrom({
          'descriptions': {
            'zh-CN': {
              'language': 'zh-CN',
              'content': '   ',
              'source': {'type': 'nextmoe', 'name': 'NextMoe 资料库'},
            },
            'en-US': {
              'language': 'en-US',
              'content': 'English',
              'source': {'type': 'vndb', 'name': 'VNDB'},
            },
          },
        }),
        'zh-CN',
      );

      expect(best!.locale, 'en-US');
      expect(best.isFallback, isTrue);
    });

    test('returns null for empty descriptions', () {
      expect(getBestDescription(const {}, 'zh-CN'), isNull);
      expect(getBestDescription({}, 'zh-CN'), isNull);
    });
  });

  group('MarkdownView', () {
    testWidgets('renders markdown content as text', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: MarkdownView(data: '# 标题\n\n这是**中文简介**', selectable: false),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('标题', findRichText: true), findsOneWidget);
      expect(find.textContaining('中文简介'), findsWidgets);
    });
  });
}
