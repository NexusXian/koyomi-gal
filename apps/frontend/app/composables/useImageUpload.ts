import { createImageService } from '~/services/image'
import type { ImageAsset, ImageCategory } from '~/types/image'

export const ALLOWED_IMAGE_MIME_TYPES = [
  'image/jpeg',
  'image/png',
  'image/webp',
  'image/avif',
  'image/gif'
] as const

export const IMAGE_CATEGORY_MAX_SIZES: Record<ImageCategory, number> = {
  avatars: 5 * 1024 * 1024,
  posts: 15 * 1024 * 1024,
  comments: 10 * 1024 * 1024,
  galgames: 20 * 1024 * 1024,
  backgrounds: 20 * 1024 * 1024,
  banners: 20 * 1024 * 1024,
  admin: 20 * 1024 * 1024
}

export interface UploadImageOptions {
  onProgress?: (percentage: number) => void
}

export interface ImageDimensions {
  width?: number
  height?: number
}

export class ImageUploadError extends Error {}

function validateImage(file: File, category: ImageCategory): void {
  if (!ALLOWED_IMAGE_MIME_TYPES.includes(file.type as (typeof ALLOWED_IMAGE_MIME_TYPES)[number])) {
    throw new ImageUploadError('仅支持 JPG、PNG、WebP、AVIF 和 GIF 格式的图片')
  }
  if (file.size <= 0) {
    throw new ImageUploadError('图片文件为空')
  }
  const maxSize = IMAGE_CATEGORY_MAX_SIZES[category]
  if (file.size > maxSize) {
    throw new ImageUploadError(`图片大小不能超过 ${Math.floor(maxSize / 1024 / 1024)}MB`)
  }
}

function getImageDimensions(file: File): Promise<ImageDimensions> {
  if (file.type === 'image/gif' || typeof createImageBitmap !== 'function') {
    return Promise.resolve({})
  }
  return createImageBitmap(file)
    .then((bitmap) => {
      const dimensions = { width: bitmap.width, height: bitmap.height }
      bitmap.close()
      return dimensions
    })
    .catch(() => ({}))
}

// fetch() cannot report upload progress; XHR gives real-time PUT progress.
export function uploadToPresignedURL(
  url: string,
  file: File,
  onProgress?: (percentage: number) => void
): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('PUT', url)
    xhr.setRequestHeader('Content-Type', file.type)

    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable && onProgress) {
        onProgress(Math.round((event.loaded / event.total) * 100))
      }
    }
    xhr.onerror = () => reject(new ImageUploadError('图片上传失败，请检查网络后重试'))
    xhr.onabort = () => reject(new ImageUploadError('图片上传已取消'))
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        onProgress?.(100)
        resolve()
        return
      }
      reject(new ImageUploadError(`图片上传失败（${xhr.status}）`))
    }

    xhr.send(file)
  })
}

export function useImageUpload() {
  const { $api } = useNuxtApp()
  const imageService = createImageService($api)

  async function uploadImage(
    file: File,
    category: ImageCategory,
    options: UploadImageOptions = {}
  ): Promise<ImageAsset> {
    validateImage(file, category)
    const dimensions = await getImageDimensions(file)

    const presign = await imageService.presign({
      filename: file.name || 'image',
      content_type: file.type,
      size: file.size,
      category
    })

    await uploadToPresignedURL(presign.upload_url, file, options.onProgress)

    return imageService.completeUpload(presign.id, dimensions)
  }

  return { uploadImage }
}
