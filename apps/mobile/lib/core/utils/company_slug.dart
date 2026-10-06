const _slugWhitespace =
    r'[\u0009-\u000D\u0020\u0085\u00A0\u1680\u2000-\u200A\u2028\u2029\u202F\u205F\u3000]';

// Dart trim also removes U+FEFF, which the backend preserves.
String normalizeCompanySlug(String value) => value
    .replaceAll(RegExp('^$_slugWhitespace+|$_slugWhitespace+\$'), '')
    .toLowerCase();

bool isValidCompanySlug(String value) {
  final slug = normalizeCompanySlug(value);
  return slug.isNotEmpty &&
      slug != '.' &&
      slug != '..' &&
      !RegExp(r'[/\\\x00-\x1F\x7F-\x9F]').hasMatch(slug);
}
