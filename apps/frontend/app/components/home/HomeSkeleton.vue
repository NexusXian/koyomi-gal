<template>
  <div class="home-skeleton" aria-label="首页内容加载中" aria-busy="true">
    <KunSkeleton class="banner-skeleton" />
    <div class="skeleton-columns">
      <section v-for="section in 2" :key="section">
        <KunSkeleton class="heading-skeleton" />
        <div class="skeleton-grid">
          <KunSkeleton v-for="item in 3" :key="item" class="card-skeleton" />
        </div>
      </section>
    </div>
    <KunSkeleton class="list-skeleton" />
  </div>
</template>

<style scoped>
.home-skeleton {
  display: flex;
  flex-direction: column;
  gap: 34px;
}

.banner-skeleton {
  width: 100%;
  max-height: 420px;
  aspect-ratio: 21 / 9;
  border-radius: 24px;
}

.skeleton-columns {
  display: grid;
  gap: 34px;
}

.heading-skeleton {
  width: 210px;
  height: 42px;
  margin-bottom: 14px;
}

.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

.card-skeleton { min-height: 260px; }
.list-skeleton { min-height: 210px; }

@media (min-width: 900px) {
  .skeleton-columns { grid-template-columns: minmax(0, 1.65fr) minmax(300px, 1fr); }
}

@media (max-width: 639px) {
  .banner-skeleton { aspect-ratio: 16 / 9; border-radius: 18px; }
  .skeleton-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .card-skeleton:last-child { display: none; }
}
</style>
