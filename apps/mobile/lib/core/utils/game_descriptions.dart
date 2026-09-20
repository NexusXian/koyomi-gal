import 'dart:ui';

import '../../models/galgame_models.dart';

const descriptionLocales = ['zh-CN', 'en-US', 'ja-JP'];

const localeLabels = {'zh-CN': '中文', 'en-US': '英文', 'ja-JP': '日文'};

const _fallbackChains = {
  'zh-CN': ['zh-CN', 'ja-JP', 'en-US'],
  'ja-JP': ['ja-JP', 'zh-CN', 'en-US'],
  'en-US': ['en-US', 'ja-JP', 'zh-CN'],
};

/// Normalizes a platform locale tag onto the stored description languages.
String normalizeLocale(Locale locale) {
  switch (locale.languageCode) {
    case 'zh':
      return 'zh-CN';
    case 'ja':
      return 'ja-JP';
    case 'en':
    default:
      return 'en-US';
  }
}

class BestDescription {
  const BestDescription({
    required this.description,
    required this.locale,
    required this.isFallback,
  });

  final GameDescription description;
  final String locale;
  final bool isFallback;
}

/// Returns the description matching [locale], falling back through the
/// shared chain; languages without usable content are skipped.
BestDescription? getBestDescription(
  Map<String, GameDescription> descriptions,
  String locale,
) {
  final chain = _fallbackChains[locale] ?? _fallbackChains['en-US']!;
  for (final language in chain) {
    final candidate = descriptions[language];
    if (candidate != null && candidate.content.trim().isNotEmpty) {
      return BestDescription(
        description: candidate,
        locale: language,
        isFallback: language != locale,
      );
    }
  }
  return null;
}
