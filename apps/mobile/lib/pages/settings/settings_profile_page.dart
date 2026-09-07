import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import '../../core/api/api_client.dart';
import '../../core/constants/domain.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';
import '../../widgets/user_widgets.dart';

class SettingsProfilePage extends ConsumerStatefulWidget {
  const SettingsProfilePage({super.key});

  @override
  ConsumerState<SettingsProfilePage> createState() =>
      _SettingsProfilePageState();
}

class _SettingsProfilePageState extends ConsumerState<SettingsProfilePage> {
  final _formKey = GlobalKey<FormState>();
  final _displayNameController = TextEditingController();
  final _bioController = TextEditingController();
  final _locationController = TextEditingController();
  final _websiteController = TextEditingController();
  final _birthdayController = TextEditingController();

  String _gender = 'undisclosed';
  bool _loading = true;
  bool _saving = false;
  bool _uploadingAvatar = false;
  String? _error;
  String? _avatarUrl;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _displayNameController.dispose();
    _bioController.dispose();
    _locationController.dispose();
    _websiteController.dispose();
    _birthdayController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final profile = await ref.read(meServiceProvider).profile();
      if (!mounted) {
        return;
      }
      setState(() {
        _displayNameController.text = profile.displayName ?? '';
        _bioController.text = profile.bio ?? '';
        _locationController.text = profile.location ?? '';
        _websiteController.text = profile.websiteUrl ?? '';
        _birthdayController.text = _dateOnly(profile.birthday);
        _gender = profile.gender ?? 'undisclosed';
        _avatarUrl = profile.avatarUrl;
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

  static String _dateOnly(String? value) {
    if (value == null) {
      return '';
    }
    return value.length >= 10 ? value.substring(0, 10) : value;
  }

  Future<void> _pickBirthday() async {
    final parsed = DateTime.tryParse('${_birthdayController.text}T00:00:00');
    final picked = await showDatePicker(
      context: context,
      initialDate: parsed ?? DateTime(2000),
      firstDate: DateTime(1900),
      lastDate: DateTime.now(),
    );
    if (picked != null) {
      setState(() {
        _birthdayController.text =
            '${picked.year}-${picked.month.toString().padLeft(2, '0')}-${picked.day.toString().padLeft(2, '0')}';
      });
    }
  }

  Future<void> _uploadAvatar() async {
    final picked = await ImagePicker()
        .pickImage(source: ImageSource.gallery, maxWidth: 1024);
    if (picked == null) {
      return;
    }
    final bytes = await File(picked.path).readAsBytes();
    final mime = picked.mimeType ?? 'image/jpeg';
    setState(() => _uploadingAvatar = true);
    try {
      final imageService = ref.read(imageServiceProvider);
      final presign = await imageService.presign(
        filename: picked.name.split('/').last.isEmpty
            ? 'avatar'
            : picked.name.split('/').last,
        contentType: mime,
        size: bytes.length,
        category: 'avatars',
      );
      await imageService.uploadToPresigned(presign.uploadUrl, bytes, mime);
      final asset = await imageService.complete(presign.id);
      final assetId = (asset['id'] as num?)?.toInt();
      await ref.read(meServiceProvider).updateAvatarAsset(assetId);
      if (mounted) {
        setState(() => _avatarUrl = asset['url'] as String?);
        showAppSnackBar(context, '头像已更新');
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    } finally {
      if (mounted) {
        setState(() => _uploadingAvatar = false);
      }
    }
  }

  Future<void> _submit() async {
    setState(() => _saving = true);
    try {
      await ref.read(meServiceProvider).updateProfile({
        'display_name': _displayNameController.text.trim(),
        'bio': _bioController.text.trim(),
        'location': _locationController.text.trim(),
        'website_url': _websiteController.text.trim(),
        'birthday': _birthdayController.text.trim(),
        'gender': _gender,
      });
      if (mounted) {
        showAppSnackBar(context, '资料已保存');
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

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('编辑资料'),
        actions: [
          TextButton(
            onPressed: _saving ? null : _submit,
            child: const Text('保存'),
          ),
        ],
      ),
      body: _loading
          ? const LoadingView()
          : _error != null
              ? ErrorView(message: _error!, onRetry: _load)
              : Form(
                  key: _formKey,
                  child: ListView(
                    padding: const EdgeInsets.all(16),
                    children: [
                      Center(
                        child: Column(
                          children: [
                            _AvatarPickerButton(
                              url: _avatarUrl,
                              uploading: _uploadingAvatar,
                              onPressed: _uploadAvatar,
                            ),
                            const SizedBox(height: 6),
                            Text(
                              '点击更换头像',
                              style: TextStyle(
                                fontSize: 12,
                                color: Theme.of(context)
                                    .colorScheme
                                    .onSurfaceVariant,
                              ),
                            ),
                          ],
                        ),
                      ),
                      const SizedBox(height: 20),
                      TextFormField(
                        controller: _displayNameController,
                        decoration: const InputDecoration(labelText: '昵称'),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _bioController,
                        decoration: const InputDecoration(labelText: '签名'),
                        minLines: 2,
                        maxLines: 4,
                      ),
                      const SizedBox(height: 12),
                      DropdownButtonFormField<String>(
                        initialValue: _gender,
                        decoration: const InputDecoration(labelText: '性别'),
                        items: [
                          for (final option in genderOptions)
                            DropdownMenuItem(
                              value: option.slug,
                              child: Text(option.label),
                            ),
                        ],
                        onChanged: (value) =>
                            setState(() => _gender = value ?? 'undisclosed'),
                      ),
                      const SizedBox(height: 12),
                      InkWell(
                        onTap: _pickBirthday,
                        borderRadius: BorderRadius.circular(10),
                        child: InputDecorator(
                          decoration:
                              const InputDecoration(labelText: '生日'),
                          child: Text(
                            _birthdayController.text.isEmpty
                                ? '选择日期'
                                : _birthdayController.text,
                          ),
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _locationController,
                        decoration:
                            const InputDecoration(labelText: '所在地'),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _websiteController,
                        decoration: const InputDecoration(labelText: '个人网站'),
                        keyboardType: TextInputType.url,
                      ),
                    ],
                  ),
                ),
    );
  }
}

class _AvatarPickerButton extends ConsumerWidget {
  const _AvatarPickerButton({
    required this.url,
    required this.uploading,
    required this.onPressed,
  });

  final String? url;
  final bool uploading;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return GestureDetector(
      onTap: uploading ? null : onPressed,
      child: Stack(
        children: [
          UserAvatar(url: url, size: 84),
          if (uploading)
            const Positioned.fill(
              child: Center(
                child: SizedBox(
                  width: 26,
                  height: 26,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              ),
            ),
          Positioned(
            right: 0,
            bottom: 0,
            child: CircleAvatar(
              radius: 14,
              backgroundColor: Theme.of(context).colorScheme.primary,
              child: const Icon(Icons.edit, size: 14, color: Colors.white),
            ),
          ),
        ],
      ),
    );
  }
}
