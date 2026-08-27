<template>
  <div v-if="siteKey && apiEndpoint" class="cap-widget-wrapper">
    <component
      :is="'cap-widget'"
      ref="widgetRef"
      :data-cap-api-endpoint="resolvedEndpoint"
      @solve="handleSolve"
      @error="handleError"
      @reset="handleReset"
    ></component>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'

const props = withDefaults(
  defineProps<{
    apiEndpoint: string
    siteKey: string
    theme?: 'light' | 'dark' | 'auto'
  }>(),
  {
    theme: 'auto'
  }
)

const emit = defineEmits<{
  (e: 'verify', token: string): void
  (e: 'expire'): void
  (e: 'error'): void
}>()

const SCRIPT_SRC = 'https://cdn.jsdelivr.net/npm/cap-widget'

const widgetRef = ref<HTMLElement | null>(null)
const scriptLoaded = ref(false)
let cachedToken: string | null = null

const resolvedEndpoint = computed(() => {
  let endpoint = props.apiEndpoint.trim()
  const siteKey = props.siteKey.trim()
  if (!endpoint) return ''

  if (!endpoint.startsWith('http://') && !endpoint.startsWith('https://')) {
    endpoint = 'https://' + endpoint
  }
  endpoint = endpoint.replace(/\/+$/, '')

  if (siteKey) {
    if (endpoint.endsWith('/' + siteKey)) {
      return endpoint + '/'
    }
    return `${endpoint}/${siteKey}/`
  }
  return endpoint + '/'
})

const loadScript = (): Promise<void> => {
  return new Promise((resolve, reject) => {
    if (typeof window !== 'undefined' && customElements.get('cap-widget')) {
      scriptLoaded.value = true
      resolve()
      return
    }

    const existingScript = document.querySelector<HTMLScriptElement>(
      `script[src*="cap-widget"]`
    )
    if (existingScript) {
      if (customElements.get('cap-widget')) {
        scriptLoaded.value = true
        resolve()
        return
      }
      existingScript.addEventListener('load', () => {
        scriptLoaded.value = true
        resolve()
      })
      existingScript.addEventListener('error', () => {
        reject(new Error('Failed to load Cap widget script'))
      })
      return
    }

    const script = document.createElement('script')
    script.src = SCRIPT_SRC
    script.async = true
    script.onload = () => {
      scriptLoaded.value = true
      resolve()
    }
    script.onerror = () => {
      reject(new Error('Failed to load Cap widget script'))
    }
    document.head.appendChild(script)
  })
}

const handleSolve = (e: Event) => {
  const customEvent = e as CustomEvent<{ token?: string }>
  const token = customEvent.detail?.token || ''
  if (token) {
    cachedToken = token
    emit('verify', token)
  }
}

const handleError = (e: Event) => {
  const customEvent = e as CustomEvent<{ message?: string }>
  console.error('[Cap] Widget error:', customEvent.detail?.message)
  cachedToken = null
  emit('error')
}

const handleReset = () => {
  cachedToken = null
  emit('expire')
}

const reset = () => {
  cachedToken = null
  if (widgetRef.value) {
    const customEl = widgetRef.value as HTMLElement & { reset?: () => void }
    if (typeof customEl.reset === 'function') {
      try {
        customEl.reset()
      } catch {
        // Ignore reset failures
      }
    }
  }
}

const verify = async (): Promise<string | null> => {
  if (cachedToken) {
    return cachedToken
  }
  if (!scriptLoaded.value) {
    try {
      await loadScript()
    } catch {
      return null
    }
  }
  // Programmatic solve fallback using Cap class if available
  const CapConstructor = (window as unknown as { Cap?: new (opts: { apiEndpoint: string }) => { solve: () => Promise<{ token?: string }> } }).Cap
  if (typeof CapConstructor === 'function' && resolvedEndpoint.value) {
    try {
      const solver = new CapConstructor({ apiEndpoint: resolvedEndpoint.value })
      const res = await solver.solve()
      if (res?.token) {
        cachedToken = res.token
        emit('verify', res.token)
        return res.token
      }
    } catch (err) {
      console.error('[Cap] Programmatic solve failed:', err)
      return null
    }
  }
  return null
}

const getToken = (): string | null => {
  return cachedToken
}

defineExpose({ verify, reset, getToken })

onMounted(async () => {
  if (!props.siteKey || !props.apiEndpoint) {
    return
  }

  try {
    await loadScript()
  } catch (error) {
    console.error('Failed to initialize Cap widget:', error)
    emit('error')
  }
})

onUnmounted(() => {
  cachedToken = null
})
</script>

<style scoped>
.cap-widget-wrapper {
  width: 100%;
  margin: 0.5rem 0;
}

.cap-widget-wrapper :deep(cap-widget) {
  display: block;
  width: 100%;
}
</style>
