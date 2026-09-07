import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';

class RegisterPage extends ConsumerStatefulWidget {
  const RegisterPage({super.key});

  @override
  ConsumerState<RegisterPage> createState() => _RegisterPageState();
}

class _RegisterPageState extends ConsumerState<RegisterPage> {
  final _formKey = GlobalKey<FormState>();
  final _usernameController = TextEditingController();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _confirmPasswordController = TextEditingController();
  final _codeController = TextEditingController();

  bool _obscure = true;
  bool _loading = false;
  bool _sending = false;
  int _countdown = 0;
  Timer? _timer;

  @override
  void dispose() {
    _timer?.cancel();
    _usernameController.dispose();
    _emailController.dispose();
    _passwordController.dispose();
    _confirmPasswordController.dispose();
    _codeController.dispose();
    super.dispose();
  }

  Future<void> _sendCode() async {
    final email = _emailController.text.trim();
    if (email.isEmpty) {
      showAppSnackBar(context, '请先输入邮箱', error: true);
      return;
    }
    setState(() => _sending = true);
    try {
      final message =
          await ref.read(authControllerProvider).sendVerificationCode(email);
      if (!mounted) {
        return;
      }
      showAppSnackBar(context, message);
      setState(() {
        _countdown = 60;
      });
      _timer = Timer.periodic(const Duration(seconds: 1), (timer) {
        if (_countdown <= 0) {
          timer.cancel();
        } else {
          setState(() => _countdown -= 1);
        }
      });
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
    if (!_formKey.currentState!.validate()) {
      return;
    }
    if (_passwordController.text != _confirmPasswordController.text) {
      showAppSnackBar(context, '两次输入的密码不一致', error: true);
      return;
    }
    setState(() => _loading = true);
    try {
      final message = await ref.read(authControllerProvider).register(
            username: _usernameController.text.trim(),
            email: _emailController.text.trim(),
            password: _passwordController.text,
            confirmPassword: _confirmPasswordController.text,
            verificationCode: _codeController.text.trim(),
          );
      if (!mounted) {
        return;
      }
      showAppSnackBar(context, message);
      context.pop();
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
      appBar: AppBar(title: const Text('注册')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              TextFormField(
                controller: _usernameController,
                decoration: const InputDecoration(
                  labelText: '用户名 *',
                  prefixIcon: Icon(Icons.person_outline),
                ),
                validator: (value) =>
                    value == null || value.trim().isEmpty ? '请输入用户名' : null,
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _emailController,
                decoration: const InputDecoration(
                  labelText: '邮箱 *',
                  prefixIcon: Icon(Icons.email_outlined),
                ),
                keyboardType: TextInputType.emailAddress,
                validator: (value) {
                  if (value == null || value.trim().isEmpty) {
                    return '请输入邮箱';
                  }
                  if (!value.contains('@')) {
                    return '邮箱格式不正确';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 16),
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    child: TextFormField(
                      controller: _codeController,
                      decoration: const InputDecoration(
                        labelText: '验证码 *',
                        prefixIcon: Icon(Icons.pin_outlined),
                      ),
                      validator: (value) =>
                          value == null || value.trim().isEmpty ? '请输入验证码' : null,
                    ),
                  ),
                  const SizedBox(width: 12),
                  SizedBox(
                    height: 48,
                    child: OutlinedButton(
                      onPressed: _sending || _countdown > 0 ? null : _sendCode,
                      child: Text(
                        _countdown > 0 ? '$_countdown s' : '获取验证码',
                        style: const TextStyle(fontSize: 13),
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _passwordController,
                obscureText: _obscure,
                decoration: const InputDecoration(
                  labelText: '密码 *（至少 8 位）',
                  prefixIcon: Icon(Icons.lock_outline),
                ),
                validator: (value) {
                  if (value == null || value.length < 8) {
                    return '密码至少 8 位';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _confirmPasswordController,
                obscureText: _obscure,
                decoration: const InputDecoration(
                  labelText: '确认密码 *',
                  prefixIcon: Icon(Icons.lock_outline),
                ),
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
                    : const Text('注册'),
              ),
              const SizedBox(height: 12),
              TextButton(
                onPressed: () => context.pop(),
                child: const Text('已有账号？去登录'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
