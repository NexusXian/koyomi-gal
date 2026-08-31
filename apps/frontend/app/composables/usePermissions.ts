import { getMePermissions } from '~/api/generated/me/me'

export function usePermissions() {
  const userStore = useUserStore()
  const permissions = useState<string[]>('me-permissions', () => [])
  const roles = useState<string[]>('me-roles', () => [])
  const loaded = useState('me-permissions-loaded', () => false)

  async function load(force = false): Promise<void> {
    if (loaded.value && !force) {
      return
    }

    if (!userStore.isAuthenticated) {
      permissions.value = []
      roles.value = []
      loaded.value = false
      return
    }

    try {
      const data = unwrapApiData(await getMePermissions())
      permissions.value = data.permissions ?? []
      roles.value = data.roles ?? []
      loaded.value = true
    } catch {
      permissions.value = []
      roles.value = []
    }
  }

  function reset(): void {
    permissions.value = []
    roles.value = []
    loaded.value = false
  }

  function has(code: string): boolean {
    return (
      roles.value.includes('super_admin') || permissions.value.includes(code)
    )
  }

  function hasAny(codes: string[]): boolean {
    return (
      roles.value.includes('super_admin') ||
      codes.some((code) => permissions.value.includes(code))
    )
  }

  return { permissions, roles, loaded, load, reset, has, hasAny }
}
