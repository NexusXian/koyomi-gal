import 'package:flutter/material.dart';

class DomainOption {
  const DomainOption(this.value, this.label, {this.slug, this.color});

  final int value;
  final String label;
  final String? slug;
  final Color? color;
}

const galgameStatusOptions = [
  DomainOption(0, '待审核'),
  DomainOption(1, '已发布'),
  DomainOption(2, '已拒绝'),
  DomainOption(3, '已隐藏'),
];

const ageRatingOptions = [
  DomainOption(0, '未分级'),
  DomainOption(1, '全年龄'),
  DomainOption(4, '12+'),
  DomainOption(2, '15+'),
  DomainOption(5, '17+'),
  DomainOption(3, '18+'),
];

const userStateOptions = [
  DomainOption(1, '想玩'),
  DomainOption(2, '在玩'),
  DomainOption(3, '玩完'),
  DomainOption(4, '搁置'),
  DomainOption(5, '弃坑'),
];

const resourceTypeOptions = [
  DomainOption(0, '其他'),
  DomainOption(1, '游戏本体'),
  DomainOption(2, '补丁'),
  DomainOption(3, '存档'),
  DomainOption(4, '原声集'),
  DomainOption(5, 'CG'),
  DomainOption(6, '攻略'),
  DomainOption(7, '官方网站'),
  DomainOption(8, '购买页面'),
  DomainOption(9, '电子书'),
  DomainOption(10, '实体书'),
  DomainOption(11, '翻译版本'),
  DomainOption(12, '资料合集'),
];

const galgameSortOptions = [
  DomainOption(0, '最新', slug: 'latest'),
  DomainOption(1, '最早', slug: 'oldest'),
  DomainOption(2, '评分最高', slug: 'rating'),
  DomainOption(3, '收藏最多', slug: 'favorite'),
  DomainOption(4, '最热门', slug: 'popular'),
];

const novelSortOptions = [
  DomainOption(0, '最近更新', slug: 'updated'),
  DomainOption(1, '最新收录', slug: 'latest'),
  DomainOption(2, '最早出版', slug: 'release_asc'),
  DomainOption(3, '最新出版', slug: 'release'),
];

const novelReleaseStatusOptions = [
  DomainOption(0, '连载中', slug: 'ongoing'),
  DomainOption(1, '已完结', slug: 'completed'),
  DomainOption(2, '休刊中', slug: 'hiatus'),
  DomainOption(3, '已放弃', slug: 'cancelled'),
  DomainOption(4, '未知', slug: 'unknown'),
];

const novelRelationTypeOptions = [
  DomainOption(0, '改编作品', slug: 'adaptation'),
  DomainOption(1, '原作', slug: 'original'),
  DomainOption(2, '衍生作品', slug: 'spin_off'),
  DomainOption(3, '续作', slug: 'sequel'),
  DomainOption(4, '前作', slug: 'prequel'),
  DomainOption(5, '同系列', slug: 'same_series'),
  DomainOption(6, '相关作品', slug: 'related'),
];

const reportReasonOptions = [
  DomainOption(0, '链接失效'),
  DomainOption(1, '密码错误'),
  DomainOption(2, '文件损坏'),
  DomainOption(3, '疑似恶意软件'),
  DomainOption(4, '版本不符'),
  DomainOption(5, '重复资源'),
  DomainOption(6, '其他'),
];

const feedbackTypeOptions = [
  DomainOption(0, '意见反馈', slug: 'feedback'),
  DomainOption(1, '版权投诉', slug: 'copyright'),
];

const genderOptions = [
  DomainOption(0, '保密', slug: 'undisclosed'),
  DomainOption(1, '男', slug: 'male'),
  DomainOption(2, '女', slug: 'female'),
  DomainOption(3, '其他', slug: 'non_binary'),
];

const profileVisibilityOptions = [
  DomainOption(0, '公开', slug: 'public'),
  DomainOption(1, '仅注册用户', slug: 'registered'),
  DomainOption(2, '私密', slug: 'private'),
];

String domainLabel(List<DomainOption> options, int? value) {
  if (value == null) {
    return '-';
  }
  return options
      .where((option) => option.value == value)
      .map((option) => option.label)
      .followedBy(['-'])
      .first;
}

String domainSlug(List<DomainOption> options, int value) {
  return options
      .where((option) => option.value == value)
      .map((option) => option.slug ?? '')
      .followedBy(['latest'])
      .first;
}

int domainValueFromSlug(List<DomainOption> options, String? slug) {
  if (slug == null) {
    return 0;
  }
  return options
      .where((option) => option.slug == slug)
      .map((option) => option.value)
      .followedBy([0])
      .first;
}

int domainValueFromSlugString(List<DomainOption> options, String? slug) {
  if (slug == null) {
    return 0;
  }
  return options
      .where((option) => option.slug == slug)
      .map((option) => option.value)
      .followedBy([0])
      .first;
}

String ageRatingLabel(int? value) => domainLabel(ageRatingOptions, value);
