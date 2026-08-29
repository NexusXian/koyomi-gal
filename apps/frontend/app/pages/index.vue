<template>
  <div>
    <h1>This is the home page</h1>
    <button @click="testPage">click me to test</button>
  </div>
</template>


<script setup lang="ts">
  function testPage() {
    navigateTo('/test');
  }
</script>
