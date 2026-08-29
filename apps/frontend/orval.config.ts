import { defineConfig } from 'orval'

export default defineConfig({
  api: {
    input: {
      target: 'http://localhost:8080/swagger/doc.json'
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
