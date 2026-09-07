import 'package:flutter/material.dart';

/// Colors mirrored from the web app's @kungal/ui-tokens OKLCH palette
/// (converted to sRGB hex; light and dark variants).
class KunPalette {
  const KunPalette._();

  // Light
  static const lightBackground = Color(0xFFF4F4F7);
  static const lightForeground = Color(0xFF11181C);
  static const lightContent1 = Color(0xFFFFFFFF);
  static const lightContent2 = Color(0xFFF4F4F5);
  static const lightContent3 = Color(0xFFE4E4E7);
  static const lightContent4 = Color(0xFFD4D4D8);
  static const lightBorder = Color(0xFFECECF7);
  static const lightBorderStrong = Color(0xFFDADAE5);
  static const lightMuted = Color(0xFF85858F);
  static const lightPrimary50 = Color(0xFFF0F6FF);
  static const lightPrimary100 = Color(0xFFE2EEFF);

  // Dark
  static const darkBackground = Color(0xFF0A0A0A);
  static const darkForeground = Color(0xFFECEDEE);
  static const darkContent1 = Color(0xFF18181B);
  static const darkContent2 = Color(0xFF27272A);
  static const darkContent3 = Color(0xFF3F3F46);
  static const darkContent4 = Color(0xFF52525B);
  static const darkBorder = Color(0xFF282830);
  static const darkBorderStrong = Color(0xFF3C3C45);
  static const darkMuted = Color(0xFF85858F);

  // Shared accents
  static const primary = Color(0xFF1271EA);
  static const primaryDark = Color(0xFF4B93F0);
  static const onPrimary = Color(0xFFFFFFFF);
  static const secondary = Color(0xFFFF94DA);
  static const onSecondary = Color(0xFF1F101A);
  static const success = Color(0xFF22C55E);
  static const warning = Color(0xFFF59E0B);
  static const danger = Color(0xFFEF4444);
}

ThemeData buildLightTheme() {
  const foreground = KunPalette.lightForeground;
  final scheme = ColorScheme.light(
    primary: KunPalette.primary,
    onPrimary: KunPalette.onPrimary,
    primaryContainer: KunPalette.lightPrimary100,
    onPrimaryContainer: KunPalette.primary,
    secondary: KunPalette.secondary,
    onSecondary: KunPalette.onSecondary,
    surface: KunPalette.lightContent1,
    onSurface: foreground,
    surfaceContainerHighest: KunPalette.lightContent2,
    onSurfaceVariant: KunPalette.lightMuted,
    error: KunPalette.danger,
    onError: Colors.white,
    outline: KunPalette.lightBorder,
    outlineVariant: KunPalette.lightBorder,
    tertiary: KunPalette.success,
  );

  return _buildTheme(
    scheme,
    scaffold: KunPalette.lightBackground,
    card: KunPalette.lightContent1,
    foreground: foreground,
    muted: KunPalette.lightMuted,
    border: KunPalette.lightBorder,
    primarySoft: KunPalette.lightPrimary50,
  );
}

ThemeData buildDarkTheme() {
  const foreground = KunPalette.darkForeground;
  final scheme = ColorScheme.dark(
    primary: KunPalette.primaryDark,
    onPrimary: KunPalette.onPrimary,
    primaryContainer: const Color(0xFF0E2A4A),
    onPrimaryContainer: const Color(0xFFBFD6FB),
    secondary: KunPalette.secondary,
    onSecondary: KunPalette.onSecondary,
    surface: KunPalette.darkContent1,
    onSurface: foreground,
    surfaceContainerHighest: KunPalette.darkContent2,
    onSurfaceVariant: KunPalette.darkMuted,
    error: KunPalette.danger,
    onError: Colors.white,
    outline: KunPalette.darkBorder,
    outlineVariant: KunPalette.darkBorder,
    tertiary: KunPalette.success,
  );

  return _buildTheme(
    scheme,
    scaffold: KunPalette.darkBackground,
    card: KunPalette.darkContent1,
    foreground: foreground,
    muted: KunPalette.darkMuted,
    border: KunPalette.darkBorder,
    primarySoft: const Color(0xFF101A26),
  );
}

ThemeData _buildTheme(
  ColorScheme scheme, {
  required Color scaffold,
  required Color card,
  required Color foreground,
  required Color muted,
  required Color border,
  required Color primarySoft,
}) {
  final base = ThemeData(useMaterial3: true, colorScheme: scheme);

  return base.copyWith(
    scaffoldBackgroundColor: scaffold,
    canvasColor: scaffold,
    cardColor: card,
    appBarTheme: AppBarTheme(
      backgroundColor: scaffold,
      foregroundColor: foreground,
      elevation: 0,
      scrolledUnderElevation: 0.5,
      centerTitle: true,
      titleTextStyle: base.textTheme.titleMedium?.copyWith(
        color: foreground,
        fontWeight: FontWeight.w600,
      ),
    ),
    cardTheme: CardThemeData(
      color: card,
      elevation: 0,
      margin: EdgeInsets.zero,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: border),
      ),
    ),
    dividerTheme: DividerThemeData(color: border, thickness: 0.5, space: 0.5),
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: card,
      indicatorColor: primarySoft,
      labelTextStyle: WidgetStatePropertyAll(
        TextStyle(fontSize: 11, color: foreground, fontWeight: FontWeight.w500),
      ),
      iconTheme: WidgetStateProperty.resolveWith(
        (states) => IconThemeData(
          color: states.contains(WidgetState.selected) ? scheme.primary : muted,
        ),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: card,
      hintStyle: TextStyle(color: muted),
      contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(10),
        borderSide: BorderSide(color: border),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(10),
        borderSide: BorderSide(color: border),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(10),
        borderSide: BorderSide(color: scheme.primary, width: 1.5),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 12),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        side: BorderSide(color: border),
        padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 12),
      ),
    ),
    chipTheme: ChipThemeData(
      side: BorderSide(color: border),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      backgroundColor: card,
      selectedColor: primarySoft,
      labelStyle: TextStyle(color: foreground, fontSize: 13),
      secondaryLabelStyle: TextStyle(color: scheme.primary, fontSize: 13),
      showCheckmark: false,
      padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 0),
    ),
    snackBarTheme: const SnackBarThemeData(behavior: SnackBarBehavior.floating),
    bottomSheetTheme: const BottomSheetThemeData(
      showDragHandle: true,
      backgroundColor: null,
    ),
    listTileTheme: ListTileThemeData(
      iconColor: foreground,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
    ),
    tabBarTheme: TabBarThemeData(
      labelColor: scheme.primary,
      unselectedLabelColor: muted,
      indicatorColor: scheme.primary,
      dividerColor: border,
      labelStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
      unselectedLabelStyle: const TextStyle(fontSize: 14),
    ),
  );
}
