import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/article_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';
import '../../widgets/markdown_view.dart';

class ArticleDetailPage extends ConsumerStatefulWidget {
  const ArticleDetailPage({super.key, required this.id});

  final int id;

  @override
  ConsumerState<ArticleDetailPage> createState() => _ArticleDetailPageState();
}

class _ArticleDetailPageState extends ConsumerState<ArticleDetailPage> {
  ArticleDetail? _article;
  String? _error;
  bool _loading = true;

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
      final article = await ref.read(articleServiceProvider).get(widget.id);
      if (!mounted) {
        return;
      }
      setState(() {
        _article = article;
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

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(),
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
    final article = _article!;
    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
      children: [
        Text(
          article.title ?? '',
          style: const TextStyle(fontSize: 19, fontWeight: FontWeight.w700),
        ),
        const SizedBox(height: 8),
        Text(
          [
            article.type ?? '',
            '发布于 ${formatDateTime(article.publishedAt)}',
            '${article.viewCount} 次浏览',
          ].join(' · '),
          style: TextStyle(
            fontSize: 12,
            color: Theme.of(context).colorScheme.onSurfaceVariant,
          ),
        ),
        const Divider(height: 20),
        if (article.coverUrl?.isNotEmpty == true) ...[
          AppImage(
            url: article.coverUrl,
            borderRadius: BorderRadius.circular(10),
          ),
          const SizedBox(height: 12),
        ],
        if (article.editorMode == 'markdown' && article.content != null)
          MarkdownView(data: article.content!)
        else
          Text(
            article.content ?? '',
            style: const TextStyle(fontSize: 15, height: 1.65),
          ),
      ],
    );
  }
}
