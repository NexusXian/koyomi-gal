import 'dart:io';

import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

import '../../core/api/api_client.dart';
import '../../providers/app_providers.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';

class VolumeFormPage extends ConsumerStatefulWidget {
  const VolumeFormPage({super.key, required this.novelId, this.volumeId});

  final int novelId;
  final int? volumeId;

  @override
  ConsumerState<VolumeFormPage> createState() => _VolumeFormPageState();
}

class _VolumeFormPageState extends ConsumerState<VolumeFormPage> {
  final _formKey = GlobalKey<FormState>();
  final _titleController = TextEditingController();
  final _originalTitleController = TextEditingController();
  final _isbnController = TextEditingController();
  final _releaseDateController = TextEditingController();
  final _summaryController = TextEditingController();
  final _volumeNumberController = TextEditingController();

  String? _coverUrl;
  bool _saving = false;
  bool _loading = false;
  String? _error;

  bool get isEditing => widget.volumeId != null;

  @override
  void initState() {
    super.initState();
    if (isEditing) {
      _loadEditing();
    }
  }

  @override
  void dispose() {
    _titleController.dispose();
    _originalTitleController.dispose();
    _isbnController.dispose();
    _releaseDateController.dispose();
    _summaryController.dispose();
    _volumeNumberController.dispose();
    super.dispose();
  }

  Future<void> _loadEditing() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final volumes =
          await ref.read(novelServiceProvider).volumes(widget.novelId);
      final volume = volumes.items
          .where((volume) => volume.id == widget.volumeId)
          .firstOrNull;
      if (!mounted) {
        return;
      }
      if (volume == null) {
        setState(() {
          _error = '未找到该卷册';
          _loading = false;
        });
        return;
      }
      setState(() {
        _titleController.text = volume.title ?? '';
        _originalTitleController.text = volume.originalTitle ?? '';
        _isbnController.text = volume.isbn ?? '';
        _releaseDateController.text = _dateOnly(volume.releaseDate);
        _summaryController.text = volume.summary ?? '';
        _volumeNumberController.text =
            volume.volumeNumber?.toString() ?? '';
        _coverUrl = volume.coverUrl;
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

  Future<void> _pickDate() async {
    final parsed = DateTime.tryParse('${_releaseDateController.text}T00:00:00');
    final picked = await showDatePicker(
      context: context,
      initialDate: parsed ?? DateTime.now(),
      firstDate: DateTime(1950),
      lastDate: DateTime(2100),
    );
    if (picked != null) {
      setState(() {
        _releaseDateController.text = DateFormat('yyyy-MM-dd').format(picked);
      });
    }
  }

  Future<void> _pickCover() async {
    final picked = await ImagePicker()
        .pickImage(source: ImageSource.gallery, maxWidth: 2400);
    if (picked == null) {
      return;
    }
    final bytes = await File(picked.path).readAsBytes();
    final mime = picked.mimeType ?? 'image/jpeg';
    setState(() => _saving = true);
    try {
      final service = ref.read(imageServiceProvider);
      final presign = await service.presign(
        filename: picked.name.split('/').last.isEmpty
            ? 'image'
            : picked.name.split('/').last,
        contentType: mime,
        size: bytes.length,
        category: 'novels',
      );
      await service.uploadToPresigned(presign.uploadUrl, bytes, mime);
      final asset = await service.complete(presign.id);
      if (mounted) {
        setState(() => _coverUrl = asset['url'] as String?);
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

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    setState(() => _saving = true);
    final payload = <String, dynamic>{
      'title': _titleController.text.trim(),
      'original_title': _originalTitleController.text.trim(),
      'isbn': _isbnController.text.trim(),
      'release_date': _releaseDateController.text.trim(),
      'summary': _summaryController.text.trim(),
      'volume_number': int.tryParse(_volumeNumberController.text.trim()) ?? 0,
      'cover_url': _coverUrl,
    };

    try {
      final service = ref.read(novelServiceProvider);
      if (isEditing) {
        payload['status'] = 1;
        await service.updateVolume(
          widget.novelId,
          widget.volumeId!,
          payload,
        );
        if (mounted) {
          showAppSnackBar(context, '已保存');
          context.pop(true);
        }
      } else {
        await service.createVolume(widget.novelId, payload);
        if (mounted) {
          showAppSnackBar(context, '已提交，等待审核');
          context.pop(true);
        }
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
        title: Text(isEditing ? '编辑卷册' : '新增卷册'),
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
              ? ErrorView(message: _error!, onRetry: _loadEditing)
              : Form(
                  key: _formKey,
                  child: ListView(
                    padding: const EdgeInsets.all(16),
                    children: [
                      TextFormField(
                        controller: _titleController,
                        decoration: const InputDecoration(labelText: '卷标题'),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _originalTitleController,
                        decoration:
                            const InputDecoration(labelText: '原文标题'),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _volumeNumberController,
                        decoration:
                            const InputDecoration(labelText: '卷号（0-9999）'),
                        keyboardType: TextInputType.number,
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _isbnController,
                        decoration: const InputDecoration(labelText: 'ISBN'),
                      ),
                      const SizedBox(height: 12),
                      InkWell(
                        onTap: _pickDate,
                        borderRadius: BorderRadius.circular(10),
                        child: InputDecorator(
                          decoration:
                              const InputDecoration(labelText: '出版日期'),
                          child: Text(
                            _releaseDateController.text.isEmpty
                                ? '选择日期'
                                : _releaseDateController.text,
                          ),
                        ),
                      ),
                      const SizedBox(height: 16),
                      Row(
                        children: [
                          AppImage(
                            url: _coverUrl,
                            width: 64,
                            height: 90,
                            borderRadius: BorderRadius.circular(8),
                          ),
                          const SizedBox(width: 12),
                          OutlinedButton.icon(
                            onPressed: _saving ? null : _pickCover,
                            icon: const Icon(Icons.upload_outlined, size: 16),
                            label: const Text('上传封面'),
                          ),
                        ],
                      ),
                      const SizedBox(height: 16),
                      TextFormField(
                        controller: _summaryController,
                        decoration:
                            const InputDecoration(labelText: '简介（支持 Markdown）'),
                        minLines: 4,
                        maxLines: 10,
                      ),
                      const SizedBox(height: 24),
                      FilledButton(
                        onPressed: _saving ? null : _submit,
                        child: _saving
                            ? const SizedBox(
                                width: 20,
                                height: 20,
                                child: CircularProgressIndicator(strokeWidth: 2),
                              )
                            : Text(isEditing ? '保存修改' : '提交'),
                      ),
                    ],
                  ),
                ),
    );
  }
}
