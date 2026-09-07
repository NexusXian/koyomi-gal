import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';

/// Network image with fallback placeholder and optional sensitive blur.
class AppImage extends StatefulWidget {
  const AppImage({
    super.key,
    this.url,
    this.sensitive = false,
    this.width,
    this.height,
    this.fit = BoxFit.cover,
    this.borderRadius,
  });

  final String? url;
  final bool sensitive;
  final double? width;
  final double? height;
  final BoxFit fit;
  final BorderRadius? borderRadius;

  @override
  State<AppImage> createState() => _AppImageState();
}

class _AppImageState extends State<AppImage> {
  bool _revealed = false;

  @override
  Widget build(BuildContext context) {
    Widget image;
    if (widget.url == null || widget.url!.isEmpty) {
      image = _placeholder(context);
    } else {
      image = CachedNetworkImage(
        imageUrl: widget.url!,
        width: widget.width,
        height: widget.height,
        fit: widget.fit,
        placeholder: (_, _) => _placeholder(context),
        errorWidget: (_, _, _) => _placeholder(context),
      );
    }

    if (widget.sensitive && !_revealed) {
      image = Stack(
        fit: StackFit.passthrough,
        children: [
          image,
          Positioned.fill(
            child: ClipRRect(
              borderRadius: widget.borderRadius ?? BorderRadius.zero,
              child: BackdropFilter(
                filter: const ColorFilter.mode(
                  Color(0x66101010),
                  BlendMode.srcOver,
                ),
                child: Container(
                  color: Colors.black12,
                  alignment: Alignment.center,
                  child: const Icon(
                    Icons.visibility_off_outlined,
                    color: Colors.white70,
                    size: 20,
                  ),
                ),
              ),
            ),
          ),
        ],
      );
    }

    if (widget.borderRadius != null) {
      image = ClipRRect(borderRadius: widget.borderRadius!, child: image);
    }

    if (widget.sensitive && !_revealed) {
      image = GestureDetector(
        onTap: () => setState(() => _revealed = true),
        child: image,
      );
    }

    return image;
  }

  Widget _placeholder(BuildContext context) {
    return Container(
      width: widget.width,
      height: widget.height,
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      child: const Center(
        child: Icon(Icons.image_outlined, size: 22, color: Colors.white24),
      ),
    );
  }
}

/// Placeholder whose colors follow the theme without image dependency.
class AppPlaceholder extends StatelessWidget {
  const AppPlaceholder({super.key, this.width, this.height});

  final double? width;
  final double? height;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: width,
      height: height,
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(8),
      ),
    );
  }
}
