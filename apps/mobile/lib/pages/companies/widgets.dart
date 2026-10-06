import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api/api_client.dart';
import '../../models/company_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';
import 'company_list_page.dart';

class CompanyPermissionView extends ConsumerWidget {
  const CompanyPermissionView({
    super.key,
    required this.title,
    required this.child,
  });

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) => ref
      .watch(mePermissionsProvider)
      .when(
        data: (permissions) => permissions.has('company:manage')
            ? child
            : Scaffold(
                appBar: AppBar(title: Text(title)),
                body: const EmptyView(hint: '无权管理公司'),
              ),
        loading: () => Scaffold(
          appBar: AppBar(title: Text(title)),
          body: const LoadingView(),
        ),
        error: (error, _) => Scaffold(
          appBar: AppBar(title: Text(title)),
          body: ErrorView(
            message: apiErrorMessage(error),
            onRetry: () => ref.invalidate(mePermissionsProvider),
          ),
        ),
      );
}

class CompanyPagination extends StatelessWidget {
  const CompanyPagination({
    super.key,
    required this.page,
    required this.total,
    required this.hasMore,
    required this.loading,
    required this.onPage,
  });

  final int page, total;
  final bool hasMore, loading;
  final ValueChanged<int> onPage;

  @override
  Widget build(BuildContext context) => SafeArea(
    top: false,
    child: Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        IconButton(
          tooltip: '上一页',
          onPressed: !loading && page > 1 ? () => onPage(page - 1) : null,
          icon: const Icon(Icons.chevron_left),
        ),
        Text('第 $page 页 · 共 $total 条'),
        IconButton(
          tooltip: '下一页',
          onPressed: !loading && hasMore ? () => onPage(page + 1) : null,
          icon: const Icon(Icons.chevron_right),
        ),
      ],
    ),
  );
}

Future<CompanySummary?> selectCompany(BuildContext context, {int? excludeId}) =>
    showModalBottomSheet<CompanySummary>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      builder: (context) => FractionallySizedBox(
        heightFactor: .9,
        child: CompanyListPage(
          excludeId: excludeId,
          onSelected: (company) => Navigator.of(context).pop(company),
        ),
      ),
    );

class GameCompanyEditor extends StatelessWidget {
  const GameCompanyEditor({
    super.key,
    required this.companies,
    required this.onChanged,
    this.enabled = true,
  });

  final List<GameCompany> companies;
  final ValueChanged<List<GameCompany>> onChanged;
  final bool enabled;

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      const Text('关联公司与角色'),
      for (var index = 0; index < companies.length; index++)
        Row(
          children: [
            Expanded(child: Text(companies[index].name)),
            DropdownButton<String>(
              value: companyRoles.containsKey(companies[index].role)
                  ? companies[index].role
                  : null,
              hint: Text(companies[index].role),
              items: [
                for (final role in companyRoles.entries)
                  DropdownMenuItem(value: role.key, child: Text(role.value)),
              ],
              onChanged: !enabled
                  ? null
                  : (role) {
                      if (role == null) return;
                      final company = companies[index];
                      if (role == company.role) return;
                      if (companies.any(
                        (other) => other.id == company.id && other.role == role,
                      )) {
                        showAppSnackBar(context, '该公司已关联此角色', error: true);
                        return;
                      }
                      final next = List<GameCompany>.of(companies);
                      next[index] = GameCompany(
                        id: company.id,
                        name: company.name,
                        slug: company.slug,
                        logoUrl: company.logoUrl,
                        role: role,
                      );
                      onChanged(next);
                    },
            ),
            IconButton(
              tooltip: '移除公司',
              onPressed: !enabled
                  ? null
                  : () => onChanged(
                      List<GameCompany>.of(companies)..removeAt(index),
                    ),
              icon: const Icon(Icons.close),
            ),
          ],
        ),
      OutlinedButton.icon(
        onPressed: !enabled
            ? null
            : () async {
                final company = await selectCompany(context);
                if (company == null || !context.mounted) return;
                final available = companyRoles.keys.where(
                  (role) => !companies.any(
                    (item) => item.id == company.id && item.role == role,
                  ),
                );
                if (available.isEmpty) {
                  showAppSnackBar(context, '该公司已关联所有角色', error: true);
                  return;
                }
                onChanged([
                  ...companies,
                  GameCompany(
                    id: company.id,
                    name: company.name,
                    slug: company.slug,
                    logoUrl: company.logoUrl,
                    role: available.first,
                  ),
                ]);
              },
        icon: const Icon(Icons.add),
        label: const Text('搜索并添加公司'),
      ),
    ],
  );
}
