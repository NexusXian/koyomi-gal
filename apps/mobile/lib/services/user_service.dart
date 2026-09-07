import '../core/api/api_client.dart';
import '../models/auth_models.dart';
import '../models/galgame_models.dart';
import '../models/pagination.dart';
import '../models/user_models.dart';

class UserService {
  UserService(this._api);

  final ApiClient _api;

  Future<PublicUserProfile> profile(String username) async {
    final data = await _api.get('/api/v1/users/$username');
    return PublicUserProfile.fromMap(Map<String, dynamic>.from(data));
  }

  Future<UserLevelData> level(String username) async {
    final data = await _api.get('/api/v1/users/$username/level');
    return UserLevelData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<Paginated<ProfilePostData>> posts(String username,
      {int page = 1, int limit = 20}) async {
    final data = await _api.get(
      '/api/v1/users/$username/posts',
      queryParameters: {'page': page, 'limit': limit},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: ProfilePostData.fromMap,
    );
  }

  Future<Paginated<ProfileCommentData>> comments(String username,
      {int page = 1, int limit = 20}) async {
    final data = await _api.get(
      '/api/v1/users/$username/comments',
      queryParameters: {'page': page, 'limit': limit},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: ProfileCommentData.fromMap,
    );
  }

  Future<Paginated<ProfileGalgameData>> ratings(String username,
      {int page = 1, int limit = 20}) async {
    final data = await _api.get(
      '/api/v1/users/$username/ratings',
      queryParameters: {'page': page, 'limit': limit},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: ProfileGalgameData.fromMap,
    );
  }

  Future<Paginated<ProfileGalgameData>> favorites(String username,
      {int page = 1, int limit = 20}) async {
    final data = await _api.get(
      '/api/v1/users/$username/favorites',
      queryParameters: {'page': page, 'limit': limit},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: ProfileGalgameData.fromMap,
    );
  }

  Future<Paginated<UserActivityData>> activities(String username,
      {int page = 1, int limit = 20}) async {
    final data = await _api.get(
      '/api/v1/users/$username/activities',
      queryParameters: {'page': page, 'limit': limit},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: UserActivityData.fromMap,
    );
  }
}

class MeService {
  MeService(this._api);

  final ApiClient _api;

  Future<MePermissions> permissions() async {
    final data = await _api.get('/api/v1/me/permissions');
    return MePermissions.fromMap(Map<String, dynamic>.from(data));
  }

  Future<PublicUserProfile> profile() async {
    final data = await _api.get('/api/v1/users/me/profile');
    return PublicUserProfile.fromMap(Map<String, dynamic>.from(data));
  }

  Future<PublicUserProfile> updateProfile(Map<String, dynamic> payload) async {
    final data = await _api.patch('/api/v1/users/me/profile', data: payload);
    return PublicUserProfile.fromMap(Map<String, dynamic>.from(data));
  }

  Future<void> updateAvatarAsset(int? avatarAssetId) => _api.patch(
        '/api/v1/me',
        data: {'avatar_asset_id': avatarAssetId},
      );

  Future<PrivacySettingsData> privacy() async {
    final data = await _api.get('/api/v1/users/me/privacy');
    return PrivacySettingsData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<PrivacySettingsData> updatePrivacy(PrivacySettingsData settings) async {
    final data = await _api.patch(
      '/api/v1/users/me/privacy',
      data: {
        'profile_visibility': settings.profileVisibility,
        'show_activity': settings.showActivity,
        'show_birthday': settings.showBirthday,
        'show_comments': settings.showComments,
        'show_favorites': settings.showFavorites,
        'show_location': settings.showLocation,
        'show_posts': settings.showPosts,
        'show_ratings': settings.showRatings,
      },
    );
    return PrivacySettingsData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<UserLevelData> experience() async {
    final data = await _api.get('/api/v1/users/me/experience');
    return UserLevelData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<Paginated<ExperienceLogData>> experienceLogs(
      {int page = 1, int limit = 20}) async {
    final data = await _api.get(
      '/api/v1/users/me/experience/logs',
      queryParameters: {'page': page, 'limit': limit},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: ExperienceLogData.fromMap,
    );
  }

  Future<CheckinStatusData> checkinStatus() async {
    final data = await _api.get('/api/v1/users/me/checkin');
    return CheckinStatusData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<CheckinResultData> checkin() async {
    final data = await _api.post('/api/v1/users/me/checkin');
    return CheckinResultData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<Paginated<GalgameListItem>> galgames({
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
