import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';

class ForgotPasswordPage extends ConsumerStatefulWidget {
  const ForgotPasswordPage({super.key});

  @override
  ConsumerState<ForgotPasswordPage> createState() =>
      _ForgotPasswordPageState();
}

enum _ForgotStep { email, code, password, done }

class _ForgotPasswordPageState extends ConsumerState<ForgotPasswordPage> {
  final _formKey = GlobalKey<FormState>();
  final _emailController = TextEditingController();
  final _codeController = TextEditingController();
  final _passwordController = TextEditingController();
  final _confirmPasswordController = TextEditingController();

  _ForgotStep _step = _ForgotStep.email;
  bool _obscure = true;
  bool _loading = false;
  bool _sending = false;
  int _countdown = 0;
  Timer? _timer;
  String _email = '';
  String? _resetToken;

  @override
  void dispose() {
    _timer?.cancel();
    _emailController.dispose();
    _codeController.dispose();
    _passwordController.dispose();
    _confirmPasswordController.dispose();
    super.dispose();
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
    final email = _emailController.text.trim().toLowerCase();
    if (email.isEmpty || !email.contains('@')) {
      showAppSnackBar(context, '请输入有效的邮箱地址', error: true);
      return;
    }
    setState(() => _sending = true);
    try {
      final message =
          await ref.read(authControllerProvider).sendForgotPasswordCode(email);
      if (!mounted) {
        return;
      }
      showAppSnackBar(context, message);
      setState(() => _email = email);
      _startCountdown();
      if (_step == _ForgotStep.email) {
        setState(() => _step = _ForgotStep.code);
      }
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

  Future<void> _verifyCode() async {
    if (_loading) {
      return;
    }
    final code = _codeController.text.trim();
    if (!RegExp(r'^\d{6}$').hasMatch(code)) {
      showAppSnackBar(context, '请输入 6 位数字验证码', error: true);
      return;
    }
    setState(() => _loading = true);
    try {
      _resetToken = await ref
          .read(authControllerProvider)
          .verifyForgotPasswordCode(_email, code);
      if (!mounted) {
        return;
      }
      setState(() => _step = _ForgotStep.password);
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

  Future<void> _resetPassword() async {
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
      await ref.read(authControllerProvider).resetForgottenPassword(
            resetToken: _resetToken ?? '',
            password: _passwordController.text,
            confirmPassword: _confirmPasswordController.text,
          );
      if (!mounted) {
        return;
      }
      _resetToken = null;
      setState(() => _step = _ForgotStep.done);
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
      appBar: AppBar(title: const Text('找回密码')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: _buildStep(),
      ),
    );
  }

  Widget _buildStep() {
    switch (_step) {
      case _ForgotStep.email:
        return _buildEmailStep();
      case _ForgotStep.code:
        return _buildCodeStep();
      case _ForgotStep.password:
        return _buildPasswordStep();
      case _ForgotStep.done:
        return _buildDoneView();
    }
  }

  Widget _stepHeader(int index, String title) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          title,
          style: Theme.of(context)
              .textTheme
              .headlineSmall
              ?.copyWith(fontWeight: FontWeight.w700),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 8),
        Text(
          '第 $index 步 / 共 3 步',
          style: TextStyle(
            fontSize: 13,
            color: Theme.of(context).colorScheme.onSurfaceVariant,
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 24),
      ],
    );
  }

  Widget _buildEmailStep() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _stepHeader(1, '验证邮箱'),
        TextFormField(
          controller: _emailController,
          decoration: const InputDecoration(
            labelText: '注册邮箱',
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
        const SizedBox(height: 24),
        FilledButton(
          onPressed: _sending ? null : _sendCode,
          child: _sending
              ? const SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Text('发送验证码'),
        ),
      ],
    );
  }

  Widget _buildCodeStep() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _stepHeader(2, '输入验证码'),
        Text(
          '验证码已发送至：${maskEmail(_email)}',
          style: TextStyle(
            fontSize: 13,
            color: Theme.of(context).colorScheme.onSurfaceVariant,
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 16),
        TextFormField(
          controller: _codeController,
          decoration: const InputDecoration(
            labelText: '验证码',
            prefixIcon: Icon(Icons.pin_outlined),
          ),
          keyboardType: TextInputType.number,
          autofocus: true,
        ),
        const SizedBox(height: 8),
        OutlinedButton(
          onPressed: _sending || _countdown > 0 ? null : _sendCode,
          child: Text(
            _countdown > 0
                ? '$_countdown 秒后可重新发送'
                : _sending
                    ? '发送中...'
                    : '重新发送验证码',
          ),
        ),
        const SizedBox(height: 16),
        FilledButton(
          onPressed: _loading ? null : _verifyCode,
          child: _loading
              ? const SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Text('验证'),
        ),
      ],
    );
  }

  Widget _buildPasswordStep() {
    return Form(
      key: _formKey,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _stepHeader(3, '设置新密码'),
          TextFormField(
            controller: _passwordController,
            obscureText: _obscure,
            decoration: const InputDecoration(
              labelText: '新密码 *（8-72 位）',
              prefixIcon: Icon(Icons.lock_outline),
            ),
            autofocus: true,
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
            onPressed: _loading ? null : _resetPassword,
            child: _loading
                ? const SizedBox(
                    width: 20,
                    height: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Text('重置密码'),
          ),
        ],
      ),
    );
  }

  Widget _buildDoneView() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const SizedBox(height: 32),
        Icon(
          Icons.check_circle_outline,
          size: 64,
          color: Theme.of(context).colorScheme.primary,
        ),
        const SizedBox(height: 16),
        Text(
          '密码重置成功',
          style: Theme.of(context)
              .textTheme
              .titleLarge
              ?.copyWith(fontWeight: FontWeight.w700),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 8),
        Text(
          '请使用新密码重新登录',
          style: TextStyle(
            fontSize: 13,
            color: Theme.of(context).colorScheme.onSurfaceVariant,
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 24),
        FilledButton(
          onPressed: () => context.go('/login'),
          child: const Text('前往登录'),
        ),
      ],
    );
  }
}
