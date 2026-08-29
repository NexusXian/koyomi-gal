<template>
  <div>
    <h1>test This is a test page</h1>
    <h1>{{count}}</h1>
    <button @click="increment()">click me to test</button>
  </div>
</template>


<script setup lang="ts">
  import { ref } from 'vue';
  const count = ref(0);
  function increment() {
    count.value++;
  }
</script>


