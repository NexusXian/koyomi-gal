import 'package:flutter_test/flutter_test.dart';

import 'package:mobile/core/constants/domain.dart';

void main() {
  test('domainLabel resolves known and unknown values', () {
    expect(domainLabel(ageRatingOptions, 3), '18+');
    expect(domainLabel(ageRatingOptions, null), '-');
    expect(domainLabel(ageRatingOptions, 99), '-');
  });

  test('domainSlug maps sort index to slug', () {
    expect(domainSlug(galgameSortOptions, 2), 'rating');
    expect(domainSlug(galgameSortOptions, 99), 'latest');
  });

  test('domainValueFromSlug maps slug back to value', () {
    expect(domainValueFromSlug(novelReleaseStatusOptions, 'completed'), 1);
    expect(domainValueFromSlug(novelReleaseStatusOptions, null), 0);
  });
}
