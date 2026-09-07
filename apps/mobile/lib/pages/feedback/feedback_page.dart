import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api/api_client.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';

class FeedbackPage extends ConsumerStatefulWidget {
  const FeedbackPage({super.key});

  @override
  ConsumerState<FeedbackPage> createState() => _FeedbackPageState();
}

class _FeedbackPageState extends ConsumerState<FeedbackPage> {
  final _formKey = GlobalKey<FormState>();
  final _contentController = TextEditingController();
  final _contactController = TextEditingController();

  String _type = 'feedback';
  bool _submitting = false;

  @override
  void dispose() {
    _contentController.dispose();
    _contactController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    setState(() => _submitting = true);
    try {
      await ref.read(feedbackServiceProvider).submit(
            type: _type,
            content: _contentController.text.trim(),
            contact: _contactController.text.trim(),
          );
      if (mounted) {
        showAppSnackBar(context, '提交成功，感谢你的反馈');
        _contentController.clear();
        _contactController.clear();
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    } finally {
      if (mounted) {
        setState(() => _submitting = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('意见反馈')),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            SegmentedButton<String>(
              segments: const [
                ButtonSegment(value: 'feedback', label: Text('意见反馈')),
                ButtonSegment(value: 'copyright', label: Text('版权投诉')),
              ],
              selected: {_type},
              onSelectionChanged: (selection) =>
                  setState(() => _type = selection.first),
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _contentController,
              decoration: const InputDecoration(
                labelText: '反馈内容 *（5-2000 字）',
                alignLabelWithHint: true,
              ),
              minLines: 6,
              maxLines: 12,
              validator: (value) {
                if (value == null || value.trim().length < 5) {
                  return '内容至少 5 个字符';
                }
                if (value.trim().length > 2000) {
                  return '内容不能超过 2000 字';
                }
                return null;
              },
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _contactController,
              decoration: const InputDecoration(
                labelText: '联系方式（可选，便于我们回复）',
              ),
            ),
            const SizedBox(height: 12),
            Text(
              _type == 'copyright'
                  ? '版权投诉请附上作品链接与权利证明说明。'
                  : '匿名提交，按 IP 每小时限 5 次。',
              style: TextStyle(
                fontSize: 12,
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 20),
            FilledButton(
              onPressed: _submitting ? null : _submit,
              child: _submitting
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('提交'),
            ),
          ],
        ),
      ),
    );
  }
}
