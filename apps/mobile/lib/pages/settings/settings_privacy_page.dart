import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api/api_client.dart';
import '../../models/user_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';

class SettingsPrivacyPage extends ConsumerStatefulWidget {
  const SettingsPrivacyPage({super.key});

  @override
  ConsumerState<SettingsPrivacyPage> createState() =>
      _SettingsPrivacyPageState();
}

class _SettingsPrivacyPageState extends ConsumerState<SettingsPrivacyPage> {
  PrivacySettingsData? _settings;
  bool _loading = true;
  bool _saving = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final settings = await ref.read(meServiceProvider).privacy();
      if (!mounted) {
        return;
      }
      setState(() {
        _settings = settings;
        _loading = false;
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _error = apiErrorMessage(error);
        _loading = false;
      });
    }
  }

  Future<void> _save() async {
    final settings = _settings;
    if (settings == null) {
      return;
    }
    setState(() => _saving = true);
    try {
      final saved =
          await ref.read(meServiceProvider).updatePrivacy(settings);
      if (mounted) {
        setState(() => _settings = saved);
        showAppSnackBar(context, '隐私设置已保存');
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    } finally {
      if (mounted) {
        setState(() => _saving = false);
      }
    }
  }

  void _update(PrivacySettingsData Function(PrivacySettingsData current) update) {
    setState(() => _settings = update(_settings!));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('隐私设置'),
        actions: [
          TextButton(
            onPressed: _saving ? null : _save,
            child: const Text('保存'),
          ),
        ],
      ),
      body: _buildBody(),
    );
  }

  Widget _buildBody() {
    if (_loading) {
      return const LoadingView();
    }
    if (_error != null) {
      return ErrorView(message: _error!, onRetry: _load);
    }
    final settings = _settings!;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Padding(
                padding: EdgeInsets.fromLTRB(16, 12, 16, 4),
                child: Text('资料可见范围'),
              ),
              RadioGroup<String>(
                groupValue: settings.profileVisibility,
                onChanged: (value) => _update(
                    (current) => current.copyWith(profileVisibility: value)),
                child: const Column(
                  children: [
                    RadioListTile<String>(
                      value: 'public',
                      title: Text('公开'),
                      contentPadding:
                          EdgeInsets.symmetric(horizontal: 8),
                    ),
                    RadioListTile<String>(
                      value: 'registered',
                      title: Text('仅注册用户'),
                      contentPadding:
                          EdgeInsets.symmetric(horizontal: 8),
                    ),
                    RadioListTile<String>(
                      value: 'private',
                      title: Text('私密'),
                      contentPadding:
                          EdgeInsets.symmetric(horizontal: 8),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        Card(
          child: Column(
            children: [
              _PrivacySwitch(
                title: '展示主页帖子',
                value: settings.showPosts,
                onChanged: (value) =>
                    _update((current) => current.copyWith(showPosts: value)),
              ),
              _PrivacySwitch(
                title: '展示评论',
                value: settings.showComments,
                onChanged: (value) =>
                    _update((current) => current.copyWith(showComments: value)),
              ),
              _PrivacySwitch(
                title: '展示评分',
                value: settings.showRatings,
                onChanged: (value) =>
                    _update((current) => current.copyWith(showRatings: value)),
              ),
              _PrivacySwitch(
                title: '展示收藏',
                value: settings.showFavorites,
                onChanged: (value) => _update(
                    (current) => current.copyWith(showFavorites: value)),
              ),
              _PrivacySwitch(
                title: '展示动态',
                value: settings.showActivity,
                onChanged: (value) => _update(
                    (current) => current.copyWith(showActivity: value)),
              ),
              _PrivacySwitch(
                title: '展示生日',
                value: settings.showBirthday,
                onChanged: (value) =>
                    _update((current) => current.copyWith(showBirthday: value)),
              ),
              _PrivacySwitch(
                title: '展示所在地',
                value: settings.showLocation,
                onChanged: (value) =>
                    _update((current) => current.copyWith(showLocation: value)),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),
        FilledButton(
          onPressed: _saving ? null : _save,
          child: _saving
              ? const SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Text('保存设置'),
        ),
      ],
    );
  }
}

class _PrivacySwitch extends StatelessWidget {
  const _PrivacySwitch({
    required this.title,
    required this.value,
    required this.onChanged,
  });

  final String title;
  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return SwitchListTile(
      title: Text(title, style: const TextStyle(fontSize: 14)),
      value: value,
      onChanged: onChanged,
    );
  }
}
