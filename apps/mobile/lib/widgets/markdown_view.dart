import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';

class MarkdownView extends StatelessWidget {
  const MarkdownView({super.key, required this.data, this.selectable = true});

  final String data;
  final bool selectable;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return MarkdownBody(
      data: data,
      selectable: selectable,
      styleSheet: MarkdownStyleSheet.fromTheme(theme).copyWith(
        p: TextStyle(fontSize: 15, height: 1.65, color: theme.colorScheme.onSurface),
        h1: theme.textTheme.titleLarge
            ?.copyWith(fontWeight: FontWeight.bold, height: 1.5),
        h2: theme.textTheme.titleLarge?.copyWith(
            fontSize: 20, fontWeight: FontWeight.bold, height: 1.5),
        h3: theme.textTheme.titleMedium
            ?.copyWith(fontWeight: FontWeight.bold, height: 1.5),
        h4: theme.textTheme.titleMedium?.copyWith(height: 1.5),
        h5: theme.textTheme.titleSmall?.copyWith(height: 1.5),
        h6: theme.textTheme.titleSmall?.copyWith(height: 1.5),
        blockquote: TextStyle(color: theme.colorScheme.onSurfaceVariant),
        blockquoteDecoration: BoxDecoration(
          border: Border(
            left: BorderSide(width: 3, color: theme.colorScheme.primary),
          ),
        ),
        code: TextStyle(
          backgroundColor: theme.colorScheme.surfaceContainerHighest,
          fontSize: 13,
          fontFamily: 'monospace',
        ),
        codeblockDecoration: BoxDecoration(
          color: theme.colorScheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(8),
        ),
        listBullet: TextStyle(fontSize: 15, color: theme.colorScheme.onSurface),
        tableHead: TextStyle(fontWeight: FontWeight.bold, fontSize: 14),
        tableBody: const TextStyle(fontSize: 14),
        tableBorder: TableBorder.all(
          color: theme.colorScheme.outlineVariant,
          width: 0.5,
        ),
        horizontalRuleDecoration: BoxDecoration(
          border: Border(
            top: BorderSide(width: 0.5, color: theme.colorScheme.outlineVariant),
          ),
        ),
        a: TextStyle(color: theme.colorScheme.primary),
      ),
      onTapLink: (text, href, title) {},
    );
  }
}
