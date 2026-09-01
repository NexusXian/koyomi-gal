interface StoredBackground {
  id: string
  blob: Blob
  createdAt: number
}

const DATABASE_NAME = 'koyomi-user-preferences'
const DATABASE_VERSION = 1
const BACKGROUND_STORE_NAME = 'backgrounds'

function openDatabase(): Promise<IDBDatabase> {
  if (!import.meta.client || !globalThis.indexedDB) {
    return Promise.reject(new Error('当前环境不支持 IndexedDB'))
  }

  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DATABASE_NAME, DATABASE_VERSION)

    request.onupgradeneeded = () => {
      const database = request.result
      if (!database.objectStoreNames.contains(BACKGROUND_STORE_NAME)) {
        database.createObjectStore(BACKGROUND_STORE_NAME, { keyPath: 'id' })
      }
    }

    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error ?? new Error('无法打开背景图片存储'))
    request.onblocked = () => reject(new Error('背景图片存储正被其他页面占用'))
  })
}

async function runTransaction<T>(
  mode: IDBTransactionMode,
  operation: (store: IDBObjectStore) => IDBRequest<T>
): Promise<T> {
  const database = await openDatabase()

  try {
    return await new Promise<T>((resolve, reject) => {
      const transaction = database.transaction(BACKGROUND_STORE_NAME, mode)
      const request = operation(transaction.objectStore(BACKGROUND_STORE_NAME))
      let result: T

      request.onsuccess = () => {
        result = request.result
      }
      request.onerror = () => reject(request.error ?? new Error('背景图片存储操作失败'))
      transaction.oncomplete = () => resolve(result)
      transaction.onabort = () => reject(transaction.error ?? new Error('背景图片存储事务已中止'))
    })
  } finally {
    database.close()
  }
}

export function useBackgroundStorage() {
  async function saveBackground(blob: Blob): Promise<string> {
    const id = crypto.randomUUID()
    const background: StoredBackground = {
      id,
      blob,
      createdAt: Date.now()
    }

    await runTransaction('readwrite', (store) => store.put(background))
    return id
  }

  async function getBackground(id: string): Promise<Blob | null> {
    const background = await runTransaction<StoredBackground | undefined>(
      'readonly',
      (store) => store.get(id)
    )

    return background?.blob instanceof Blob ? background.blob : null
  }

  async function deleteBackground(id: string): Promise<void> {
    await runTransaction('readwrite', (store) => store.delete(id))
  }

  async function clearBackgrounds(): Promise<void> {
    await runTransaction('readwrite', (store) => store.clear())
  }

  return {
    saveBackground,
    getBackground,
    deleteBackground,
    clearBackgrounds
  }
}
