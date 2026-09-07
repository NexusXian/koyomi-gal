import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';

class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});

  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends ConsumerState<LoginPage> {
  final _formKey = GlobalKey<FormState>();
  final _accountController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _obscure = true;
  bool _loading = false;

  @override
  void dispose() {
    _accountController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    setState(() => _loading = true);
    try {
      await ref.read(authControllerProvider).login(
            _accountController.text.trim(),
            _passwordController.text,
          );
      if (!mounted) {
        return;
      }
      showAppSnackBar(context, '登录成功');
      final redirect = GoRouterState.of(context).uri.queryParameters['redirect'];
      if (redirect != null && redirect.isNotEmpty) {
        context.go(redirect);
      } else {
        context.go('/home');
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    } finally {
      if (mounted) {
        setState(() => _loading = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('登录')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Form(
            key: _formKey,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  '欢迎回来',
                  style: Theme.of(context)
                      .textTheme
                      .headlineSmall
                      ?.copyWith(fontWeight: FontWeight.w700),
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 24),
                TextFormField(
                  controller: _accountController,
                  decoration: const InputDecoration(
                    labelText: '用户名或邮箱',
                    prefixIcon: Icon(Icons.person_outline),
                  ),
                  validator: (value) =>
                      value == null || value.trim().isEmpty ? '请输入账号' : null,
                ),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _passwordController,
                  obscureText: _obscure,
                  decoration: InputDecoration(
                    labelText: '密码',
                    prefixIcon: const Icon(Icons.lock_outline),
                    suffixIcon: IconButton(
                      icon: Icon(
                        _obscure ? Icons.visibility_off : Icons.visibility,
                        size: 20,
                      ),
                      onPressed: () => setState(() => _obscure = !_obscure),
                    ),
                  ),
                  validator: (value) =>
                      value == null || value.isEmpty ? '请输入密码' : null,
                  onFieldSubmitted: (_) => _submit(),
                ),
                const SizedBox(height: 24),
                FilledButton(
                  onPressed: _loading ? null : _submit,
                  child: _loading
                      ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Text('登录'),
                ),
                const SizedBox(height: 12),
                TextButton(
                  onPressed: () => context.push('/register'),
                  child: const Text('没有账号？去注册'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
