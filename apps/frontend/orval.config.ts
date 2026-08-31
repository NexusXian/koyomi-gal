import { defineConfig } from 'orval'

export default defineConfig({
  api: {
    input: {
      target: 'http://localhost:8080/swagger/doc.json',
      // swag only emits Swagger 2.0, which orval 8 rejects under OpenAPI 3 validation
      unsafeDisableValidation: true
    },
    output: {
      target: './app/api/generated/endpoints.ts',
      schemas: './app/api/generated/models',
      mode: 'tags-split',
      client: 'fetch',
      override: {
        mutator: {
          path: './app/api/mutator.ts',
          name: 'apiMutator'
        },
        fetch: {
          includeHttpResponseReturnType: false
        }
      }
    }
  }
})
