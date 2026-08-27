<template>
  <div v-if="siteKey && apiEndpoint" class="cap-widget-wrapper">
    <div ref="containerRef" class="cap-widget-container"></div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

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

const containerRef = ref<HTMLElement | null>(null)
let widgetEl: HTMLElement | null = null
const scriptLoaded = ref(false)

let cachedToken: string | null = null
let pending: { resolve: (token: string | null) => void } | null = null

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
    script.type = 'module'
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
    if (pending) {
      const current = pending
      pending = null
      current.resolve(token)
    }
  }
}

const handleError = (e: Event) => {
  const customEvent = e as CustomEvent<{ message?: string }>
  console.error('[Cap] Widget error:', customEvent.detail?.message)
  emit('error')
  if (pending) {
    const current = pending
    pending = null
    current.resolve(null)
  }
}

const handleReset = () => {
  cachedToken = null
  emit('expire')
}

const attachWidgetListeners = (element: HTMLElement | null) => {
  if (!element) return
  element.addEventListener('solve', handleSolve)
  element.addEventListener('error', handleError)
  element.addEventListener('reset', handleReset)
}

const detachWidgetListeners = (element: HTMLElement | null) => {
  if (!element) return
  element.removeEventListener('solve', handleSolve)
  element.removeEventListener('error', handleError)
  element.removeEventListener('reset', handleReset)
}

const renderWidget = () => {
  if (!containerRef.value || !scriptLoaded.value || !resolvedEndpoint.value) {
    return
  }

  if (widgetEl) {
    detachWidgetListeners(widgetEl)
    widgetEl.remove()
    widgetEl = null
  }

  const el = document.createElement('cap-widget')
  el.setAttribute('data-cap-api-endpoint', resolvedEndpoint.value)
  attachWidgetListeners(el)
  containerRef.value.appendChild(el)
  widgetEl = el
}

const reset = () => {
  cachedToken = null
  if (pending) {
    const current = pending
    pending = null
    current.resolve(null)
  }
  if (widgetEl) {
    const customEl = widgetEl as HTMLElement & { reset?: () => void }
    if (typeof customEl.reset === 'function') {
      try {
        customEl.reset()
      } catch {
        // Ignore reset failures on custom elements
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
      await nextTick()
      renderWidget()
    } catch {
      return null
    }
  }
  return new Promise<string | null>((resolve) => {
    pending = { resolve }
    if (widgetEl) {
      const customEl = widgetEl as HTMLElement & { solve?: () => Promise<{ token?: string }> }
      if (typeof customEl.solve === 'function') {
        customEl
          .solve()
          .then((res) => {
            if (res?.token) {
              cachedToken = res.token
              emit('verify', res.token)
              if (pending) {
                const current = pending
                pending = null
                current.resolve(res.token)
              }
            }
          })
          .catch(() => {
            if (pending) {
              const current = pending
              pending = null
              current.resolve(null)
            }
          })
      }
    }
  })
}

defineExpose({ verify, reset })

watch([scriptLoaded, resolvedEndpoint, containerRef], () => {
  renderWidget()
})

onMounted(async () => {
  if (!props.siteKey || !props.apiEndpoint) {
    return
  }

  try {
    await loadScript()
    await nextTick()
    renderWidget()
  } catch (error) {
    console.error('Failed to initialize Cap widget:', error)
    emit('error')
  }
})

onUnmounted(() => {
  if (widgetEl) {
    detachWidgetListeners(widgetEl)
    widgetEl.remove()
    widgetEl = null
  }
  if (pending) {
    const current = pending
    pending = null
    current.resolve(null)
  }
})
</script>

<style scoped>
.cap-widget-wrapper {
  width: 100%;
}

.cap-widget-container {
  width: 100%;
  min-height: 65px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.cap-widget-container :deep(cap-widget) {
  width: 100%;
  max-width: 100%;
}
</style>
