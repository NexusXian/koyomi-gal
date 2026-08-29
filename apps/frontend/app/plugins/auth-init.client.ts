export default defineNuxtPlugin(() => {
  const { initialize } = useAuth()
  void initialize()
})
