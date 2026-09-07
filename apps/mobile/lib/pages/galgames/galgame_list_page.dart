import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/constants/domain.dart';
import '../../models/galgame_models.dart';
import '../../providers/app_providers.dart';
import '../../services/galgame_service.dart'
    show DeveloperDataLite, TagDataLite;
import '../../widgets/common_views.dart';
import 'widgets.dart';

class GalgameListPage extends ConsumerStatefulWidget {
  const GalgameListPage({super.key});

  @override
  ConsumerState<GalgameListPage> createState() => _GalgameListPageState();
}

class _GalgameListPageState extends ConsumerState<GalgameListPage> {
  final _scrollController = ScrollController();
  final _searchController = TextEditingController();
  Timer? _debounce;

  List<GalgameListItem> _items = [];
  int _total = 0;
  int _page = 1;
  bool _loading = false;
  bool _hasMore = true;
  String? _error;

  String _keyword = '';
  int _sortIndex = 0;
  int? _developerId;
  int? _ageRating;
  final List<int> _tagIds = [];

  List<TagDataLite> _tags = [];
  List<DeveloperDataLite> _developers = [];

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
    _loadFilters();
    _reload();
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _scrollController.dispose();
    _searchController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels >=
            _scrollController.position.maxScrollExtent - 200 &&
        !_loading &&
        _hasMore) {
      _loadMore();
    }
  }

  Future<void> _loadFilters() async {
    try {
      final results = await Future.wait([
        ref.read(tagServiceProvider).list(),
        ref.read(developerServiceProvider).list(),
      ]);
      if (!mounted) {
        return;
      }
      setState(() {
        _tags = results[0] as List<TagDataLite>;
        _developers = results[1] as List<DeveloperDataLite>;
      });
    } catch (_) {}
  }

  Future<void> _reload() async {
    setState(() {
      _page = 1;
      _hasMore = true;
    });
    await _loadPage(reset: true);
  }

  Future<void> _loadMore() => _loadPage();

  Future<void> _loadPage({bool reset = false}) async {
    if (_loading) {
      return;
    }
    setState(() {
      _loading = true;
      if (reset) {
        _error = null;
      }
    });
    try {
      final result = await ref.read(galgameServiceProvider).list(
            keyword: _keyword.isEmpty ? null : _keyword,
            developerId: _developerId,
            tagIds: _tagIds,
            ageRating: _ageRating,
            sort: domainSlug(galgameSortOptions, _sortIndex),
            page: _page,
            limit: 20,
          );
      if (!mounted) {
        return;
      }
      setState(() {
        _items = reset ? result.items : [..._items, ...result.items];
        _total = result.total;
        _hasMore = result.hasMore;
        _page = reset ? 2 : _page + 1;
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

  void _onSearchChanged(String value) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 350), () {
      if (_keyword != value) {
        _keyword = value;
        _reload();
      }
    });
  }

  void _openFilterSheet() {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (context) => DraggableScrollableSheet(
        initialChildSize: 0.7,
        maxChildSize: 0.9,
        minChildSize: 0.5,
        expand: false,
        builder: (context, scrollController) => _FilterSheet(
          tags: _tags,
          developers: _developers,
          selectedTagIds: List.of(_tagIds),
          selectedDeveloperId: _developerId,
          selectedAgeRating: _ageRating,
          sortIndex: _sortIndex,
          onChanged: (tagIds, developerId, ageRating, sortIndex) {
            setState(() {
              _tagIds
                ..clear()
                ..addAll(tagIds);
              _developerId = developerId;
              _ageRating = ageRating;
              _sortIndex = sortIndex;
            });
            _reload();
          },
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final hasFilter =
        _developerId != null || _ageRating != null || _tagIds.isNotEmpty;
    return Scaffold(
      appBar: AppBar(
        title: const Text('Galgame'),
        actions: [
          IconButton(
            icon: const Icon(Icons.filter_list),
            onPressed: _openFilterSheet,
          ),
        ],
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(56),
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _searchController,
                    onChanged: _onSearchChanged,
                    decoration: InputDecoration(
                      isDense: true,
                      hintText: '搜索标题或别名',
                      prefixIcon: const Icon(Icons.search, size: 20),
                      suffixIcon: _keyword.isEmpty
                          ? null
                          : IconButton(
                              icon: const Icon(Icons.close, size: 18),
                              onPressed: () {
                                _searchController.clear();
                                _keyword = '';
                                _reload();
                              },
                            ),
                    ),
                  ),
                ),
                if (hasFilter)
                  TextButton(
                    onPressed: () {
                      setState(() {
                        _tagIds.clear();
                        _developerId = null;
                        _ageRating = null;
                      });
                      _reload();
                    },
                    child: const Text('清除', style: TextStyle(fontSize: 13)),
                  ),
              ],
            ),
          ),
        ),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () => context.push('/galgames/new'),
        child: const Icon(Icons.add),
      ),
      body: _buildList(),
    );
  }

  Widget _buildList() {
    if (_error != null && _items.isEmpty) {
      return ErrorView(message: _error!, onRetry: _reload);
    }
    if (_items.isEmpty && _loading) {
      return const LoadingView();
    }
    if (_items.isEmpty) {
      return const EmptyView(hint: '没有找到 Galgame');
    }
    return RefreshIndicator(
      onRefresh: _reload,
      child: ListView.separated(
        controller: _scrollController,
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        itemCount: _items.length + 1,
        separatorBuilder: (_, _) => const SizedBox(height: 10),
        itemBuilder: (context, index) {
          if (index >= _items.length) {
            return Padding(
              padding: const EdgeInsets.symmetric(vertical: 12),
              child: Center(
                child: _loading
                    ? const SizedBox(
                        width: 22,
                        height: 22,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : _hasMore
                        ? const SizedBox.shrink()
                        : Text(
                            '共 $_total 条',
                            style: TextStyle(
                              fontSize: 12,
                              color:
                                  Theme.of(context).colorScheme.onSurfaceVariant,
                            ),
                          ),
              ),
            );
          }
          final galgame = _items[index];
          return GalgameCard(
            galgame: galgame,
            onTap: galgame.id == null
                ? null
                : () => context.push('/galgames/${galgame.id}'),
          );
        },
      ),
    );
  }
}

class _FilterSheet extends StatefulWidget {
  const _FilterSheet({
    required this.tags,
    required this.developers,
    required this.selectedTagIds,
    required this.selectedDeveloperId,
    required this.selectedAgeRating,
    required this.sortIndex,
    required this.onChanged,
  });

  final List<TagDataLite> tags;
  final List<DeveloperDataLite> developers;
  final List<int> selectedTagIds;
  final int? selectedDeveloperId;
  final int? selectedAgeRating;
  final int sortIndex;
  final void Function(
    List<int> tagIds,
    int? developerId,
    int? ageRating,
    int sortIndex,
  ) onChanged;

  @override
  State<_FilterSheet> createState() => _FilterSheetState();
}

class _FilterSheetState extends State<_FilterSheet> {
  late final List<int> _tagIds = List.of(widget.selectedTagIds);
  late int? _developerId = widget.selectedDeveloperId;
  late int? _ageRating = widget.selectedAgeRating;
  late int _sortIndex = widget.sortIndex;
  late List<TagDataLite> _filteredTags = widget.tags;
  final _tagSearchController = TextEditingController();

  @override
  void dispose() {
    _tagSearchController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 0, 20, 24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                '筛选',
                style: TextStyle(fontSize: 17, fontWeight: FontWeight.w600),
              ),
              TextButton(
                onPressed: () => Navigator.of(context).pop(),
                child: const Text('取消'),
              ),
            ],
          ),
          const SizedBox(height: 4),
          const Text('排序', style: TextStyle(fontSize: 13)),
          const SizedBox(height: 6),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (var i = 0; i < galgameSortOptions.length; i++)
                ChoiceChip(
                  label: Text(galgameSortOptions[i].label),
                  selected: _sortIndex == i,
                  onSelected: (_) => setState(() => _sortIndex = i),
                ),
            ],
          ),
          const SizedBox(height: 12),
          const Text('年龄等级', style: TextStyle(fontSize: 13)),
          const SizedBox(height: 6),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final option in ageRatingOptions)
                ChoiceChip(
                  label: Text(option.label),
                  selected: _ageRating == option.value,
                  onSelected: (_) => setState(
                    () => _ageRating =
                        _ageRating == option.value ? null : option.value,
                  ),
                ),
            ],
          ),
          const SizedBox(height: 12),
          const Text('开发商', style: TextStyle(fontSize: 13)),
          const SizedBox(height: 6),
          DropdownButtonFormField<int?>(
            initialValue: _developerId,
            decoration: const InputDecoration(isDense: true),
            items: [
              const DropdownMenuItem(value: null, child: Text('全部开发商')),
              ...widget.developers.map(
                (developer) => DropdownMenuItem(
                  value: developer.id,
                  child: Text(developer.name ?? ''),
                ),
              ),
            ],
            onChanged: (value) => setState(() => _developerId = value),
          ),
          const SizedBox(height: 12),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text('Tag（多选，AND）', style: TextStyle(fontSize: 13)),
              Text(
                '已选 ${_tagIds.length}',
                style: TextStyle(
                  fontSize: 12,
                  color: Theme.of(context).colorScheme.primary,
                ),
              ),
            ],
          ),
          const SizedBox(height: 6),
          TextField(
            controller: _tagSearchController,
            decoration: const InputDecoration(
              isDense: true,
              hintText: '搜索 Tag',
            ),
            onChanged: (value) {
              setState(() {
                _filteredTags = widget.tags
                    .where((tag) => (tag.name ?? '')
                        .toLowerCase()
                        .contains(value.toLowerCase()))
                    .toList();
              });
            },
          ),
          const SizedBox(height: 8),
          Expanded(
            child: SingleChildScrollView(
              child: Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  for (final tag in _filteredTags)
                    FilterChip(
                      label: Text(tag.name ?? ''),
                      selected: _tagIds.contains(tag.id),
                      onSelected: (selected) {
                        setState(() {
                          if (selected) {
                            _tagIds.add(tag.id!);
                          } else {
                            _tagIds.remove(tag.id);
                          }
                        });
                      },
                    ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 12),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: () {
                widget.onChanged(_tagIds, _developerId, _ageRating, _sortIndex);
                Navigator.of(context).pop();
              },
              child: const Text('应用筛选'),
            ),
          ),
        ],
      ),
    );
  }
}
