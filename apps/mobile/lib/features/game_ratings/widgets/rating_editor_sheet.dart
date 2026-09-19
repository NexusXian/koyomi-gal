import 'package:flutter/material.dart';

import '../../../widgets/markdown_editor.dart';
import '../models/game_rating_models.dart';

class RatingEditorResult {
  const RatingEditorResult.save(this.draft) : delete = false;
  const RatingEditorResult.delete() : draft = null, delete = true;

  final GameRatingDraft? draft;
  final bool delete;
}

Future<RatingEditorResult?> showRatingEditorSheet(
  BuildContext context, {
  GameRating? current,
}) {
  return showModalBottomSheet<RatingEditorResult>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    builder: (_) => RatingEditorSheet(current: current),
  );
}

class RatingEditorSheet extends StatefulWidget {
  const RatingEditorSheet({super.key, this.current});

  final GameRating? current;

  @override
  State<RatingEditorSheet> createState() => _RatingEditorSheetState();
}

class _RatingEditorSheetState extends State<RatingEditorSheet> {
  late int? _score = widget.current?.score;
  late final Map<RatingDimension, int?> _dimensions = {
    for (final dimension in RatingDimension.values)
      dimension: widget.current?.dimensions.valueOf(dimension),
  };
  late int? _recommendation = widget.current?.recommendation;
  late int _spoilerLevel = widget.current?.spoilerLevel ?? 0;
  late String _editorMode = widget.current?.editorMode == 'markdown'
      ? 'markdown'
      : 'plain';
  late final TextEditingController _reviewController = TextEditingController(
    text: widget.current?.review ?? '',
  );
  String? _error;

  @override
  void dispose() {
    _reviewController.dispose();
    super.dispose();
  }

  GameRatingDraft _draft() => GameRatingDraft(
    score: _score,
    dimensions: RatingDimensions(
      visual: _dimensions[RatingDimension.visual],
      story: _dimensions[RatingDimension.story],
      music: _dimensions[RatingDimension.music],
      character: _dimensions[RatingDimension.character],
      branch: _dimensions[RatingDimension.branch],
      system: _dimensions[RatingDimension.system],
      voice: _dimensions[RatingDimension.voice],
      replay: _dimensions[RatingDimension.replay],
    ),
    recommendation: _recommendation,
    spoilerLevel: _spoilerLevel,
    review: _reviewController.text,
    editorMode: _editorMode,
  );

  void _submit() {
    final draft = _draft();
    final error = draft.validate();
    if (error != null) {
      setState(() => _error = error);
      return;
    }
    Navigator.of(context).pop(RatingEditorResult.save(draft));
  }

  @override
  Widget build(BuildContext context) {
    final bottom = MediaQuery.viewInsetsOf(context).bottom;
    return DraggableScrollableSheet(
      expand: false,
      initialChildSize: 0.9,
      minChildSize: 0.55,
      maxChildSize: 0.96,
      builder: (context, scrollController) => AnimatedPadding(
        duration: const Duration(milliseconds: 150),
        padding: EdgeInsets.only(bottom: bottom),
        child: ListView(
          controller: scrollController,
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    widget.current == null ? '发表评分' : '编辑评分',
                    style: Theme.of(context).textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.w700),
                  ),
                ),
                if (widget.current != null)
                  TextButton.icon(
                    onPressed: () =>
                        Navigator.of(context)
                            .pop(const RatingEditorResult.delete()),
                    icon: const Icon(Icons.delete_outline, size: 18),
                    label: const Text('删除'),
                    style: TextButton.styleFrom(
                      foregroundColor: Theme.of(context).colorScheme.error,
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 12),
            const _FieldTitle(label: '综合评分', required: true),
            const SizedBox(height: 8),
            _ScorePicker(
              value: _score,
              onChanged: (value) => setState(() => _score = value),
            ),
            const SizedBox(height: 12),
            Card(
              child: ExpansionTile(
                initiallyExpanded: false,
                tilePadding: const EdgeInsets.symmetric(horizontal: 12),
                title: const Text('维度评分（可选）', style: TextStyle(fontSize: 14)),
                subtitle: const Text(
                  '展开填写画面、剧情、音乐、角色等 8 个维度',
                  style: TextStyle(fontSize: 12),
                ),
                children: [
                  for (final dimension in RatingDimension.values)
                    Padding(
                      padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            children: [
                              Expanded(child: Text(dimension.label)),
                              if (_dimensions[dimension] != null)
                                TextButton(
                                  onPressed: () => setState(
                                    () => _dimensions[dimension] = null,
                                  ),
                                  child: const Text('清除'),
                                ),
                            ],
                          ),
                          _ScorePicker(
                            value: _dimensions[dimension],
                            compact: true,
                            onChanged: (value) =>
                                setState(() => _dimensions[dimension] = value),
                          ),
                        ],
                      ),
                    ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            const _FieldTitle(label: '推荐度'),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                for (final item in RatingRecommendation.values)
                  ChoiceChip(
                    label: Text(item.label),
                    selected: _recommendation == item.value,
                    onSelected: (selected) => setState(
                      () => _recommendation = selected ? item.value : null,
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 16),
            const _FieldTitle(label: '剧透等级'),
            const SizedBox(height: 8),
            SegmentedButton<int>(
              segments: [
                for (final level in RatingSpoilerLevel.values)
                  ButtonSegment(value: level.value, label: Text(level.label)),
              ],
              selected: {_spoilerLevel},
              onSelectionChanged: (value) =>
                  setState(() => _spoilerLevel = value.first),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                const Expanded(child: _FieldTitle(label: '评价')),
                SegmentedButton<String>(
                  segments: const [
                    ButtonSegment(value: 'plain', label: Text('纯文本')),
                    ButtonSegment(value: 'markdown', label: Text('Markdown')),
                  ],
                  selected: {_editorMode},
                  onSelectionChanged: (value) =>
                      setState(() => _editorMode = value.first),
                  style: const ButtonStyle(
                    visualDensity: VisualDensity.compact,
                    textStyle: WidgetStatePropertyAll(TextStyle(fontSize: 11)),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            if (_editorMode == 'markdown')
              MarkdownEditor(
                controller: _reviewController,
                imageCategory: 'posts',
                minLines: 4,
              )
            else
              TextField(
                controller: _reviewController,
                minLines: 4,
                maxLines: 12,
                decoration: const InputDecoration(hintText: '写下你的游玩感受（可选）'),
              ),
            if (_error != null)
              Padding(
                padding: const EdgeInsets.only(top: 10),
                child: Text(
                  _error!,
                  style: TextStyle(
                    color: Theme.of(context).colorScheme.error,
                    fontSize: 12,
                  ),
                ),
              ),
            const SizedBox(height: 20),
            FilledButton(onPressed: _submit, child: const Text('保存评分')),
          ],
        ),
      ),
    );
  }
}

class _FieldTitle extends StatelessWidget {
  const _FieldTitle({required this.label, this.required = false});

  final String label;
  final bool required;

  @override
  Widget build(BuildContext context) {
    return Text.rich(
      TextSpan(
        children: [
          TextSpan(text: label),
          if (required)
            TextSpan(
              text: ' *',
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
        ],
      ),
      style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
    );
  }
}

class _ScorePicker extends StatelessWidget {
  const _ScorePicker({
    required this.value,
    required this.onChanged,
    this.compact = false,
  });

  final int? value;
  final ValueChanged<int> onChanged;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
        children: [
          for (var score = 1; score <= 10; score++)
            Padding(
              padding: const EdgeInsets.only(right: 6),
              child: ChoiceChip(
                label: Text('$score'),
                selected: value == score,
                onSelected: (_) => onChanged(score),
                visualDensity: compact ? VisualDensity.compact : null,
              ),
            ),
        ],
      ),
    );
  }
}
