import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';

class SettingsSecurityPage extends ConsumerStatefulWidget {
  const SettingsSecurityPage({super.key});

  @override
  ConsumerState<SettingsSecurityPage> createState() =>
      _SettingsSecurityPageState();
}

class _SettingsSecurityPageState extends ConsumerState<SettingsSecurityPage> {
  final _formKey = GlobalKey<FormState>();
  final _codeController = TextEditingController();
  final _passwordController = TextEditingController();
  final _confirmPasswordController = TextEditingController();

  bool _obscure = true;
  bool _loading = false;
  bool _sending = false;
  int _countdown = 0;
  Timer? _timer;
  String? _maskedEmail;

  @override
  void dispose() {
    _timer?.cancel();
    _codeController.dispose();
    _passwordController.dispose();
    _confirmPasswordController.dispose();
    super.dispose();
  }

  String get _boundEmail {
    final masked = _maskedEmail;
    if (masked != null && masked.isNotEmpty) {
      return masked;
    }
    return maskEmail(ref.read(authControllerProvider).user?.email);
  }

  void _startCountdown() {
    _timer?.cancel();
    setState(() => _countdown = 60);
    _timer = Timer.periodic(const Duration(seconds: 1), (timer) {
      if (!mounted) {
        timer.cancel();
        return;
      }
      if (_countdown <= 0) {
        timer.cancel();
      } else {
        setState(() => _countdown -= 1);
      }
    });
  }

  Future<void> _sendCode() async {
    if (_sending || _countdown > 0) {
      return;
    }
    setState(() => _sending = true);
    try {
      final email = await ref.read(authControllerProvider).sendChangePasswordCode();
      if (!mounted) {
        return;
      }
      setState(() {
        if (email.isNotEmpty) {
          _maskedEmail = email;
        }
      });
      showAppSnackBar(context, '验证码已发送，请查收邮箱');
      _startCountdown();
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    } finally {
      if (mounted) {
        setState(() => _sending = false);
      }
    }
  }

  Future<void> _submit() async {
    if (_loading) {
      return;
    }
    if (!_formKey.currentState!.validate()) {
      return;
    }
    if (_passwordController.text != _confirmPasswordController.text) {
      showAppSnackBar(context, '两次输入的密码不一致', error: true);
      return;
    }
    setState(() => _loading = true);
    try {
      await ref.read(authControllerProvider).changePassword(
            code: _codeController.text.trim(),
            newPassword: _passwordController.text,
            confirmPassword: _confirmPasswordController.text,
          );
      if (!mounted) {
        return;
      }
      showAppSnackBar(context, '密码修改成功，请重新登录');
      await ref.read(authControllerProvider).logout();
      if (!mounted) {
        return;
      }
      context.go('/login');
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
      appBar: AppBar(title: const Text('账号与安全')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Row(
                    children: [
                      const Icon(Icons.email_outlined),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const Text(
                              '修改密码',
                              style: TextStyle(fontWeight: FontWeight.w600),
                            ),
                            const SizedBox(height: 4),
                            Text(
                              _boundEmail.isEmpty
                                  ? '验证码将发送到当前账号绑定的邮箱'
                                  : '验证码将发送至：$_boundEmail',
                              style: TextStyle(
                                fontSize: 13,
                                color:
                                    Theme.of(context).colorScheme.onSurfaceVariant,
                              ),
                            ),
                          ],
                        ),
                      ),
                      const SizedBox(width: 12),
                      OutlinedButton(
                        onPressed: _sending || _countdown > 0 ? null : _sendCode,
                        child: _sending
                            ? const SizedBox(
                                width: 16,
                                height: 16,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                ),
                              )
                            : Text(
                                _countdown > 0 ? '$_countdown s' : '获取验证码',
                                style: const TextStyle(fontSize: 13),
                              ),
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _codeController,
                decoration: const InputDecoration(
                  labelText: '邮箱验证码 *',
                  prefixIcon: Icon(Icons.pin_outlined),
                ),
                keyboardType: TextInputType.number,
                validator: (value) =>
                    value == null || value.trim().isEmpty ? '请输入验证码' : null,
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _passwordController,
                obscureText: _obscure,
                decoration: const InputDecoration(
                  labelText: '新密码 *（8-72 位）',
                  prefixIcon: Icon(Icons.lock_outline),
                ),
                validator: (value) {
                  if (value == null || value.length < 8 || value.length > 72) {
                    return '密码长度必须为 8 到 72 位';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _confirmPasswordController,
                obscureText: _obscure,
                decoration: const InputDecoration(
                  labelText: '确认新密码 *',
                  prefixIcon: Icon(Icons.lock_outline),
                ),
                validator: (value) =>
                    value == null || value.isEmpty ? '请再次输入新密码' : null,
              ),
              const SizedBox(height: 8),
              Align(
                alignment: Alignment.centerRight,
                child: TextButton(
                  onPressed: () => setState(() => _obscure = !_obscure),
                  child: Text(
                    _obscure ? '显示密码' : '隐藏密码',
                    style: const TextStyle(fontSize: 12),
                  ),
                ),
              ),
              const SizedBox(height: 8),
              FilledButton(
                onPressed: _loading ? null : _submit,
                child: _loading
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('确认修改'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
