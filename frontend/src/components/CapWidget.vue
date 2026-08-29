<template>
  <div v-if="siteKey && apiEndpoint" class="cap-widget-wrapper">
    <div ref="containerRef" class="cap-widget-container"></div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, nextTick } from 'vue'
import 'cap-widget'
import { type CapWidget as CapWidgetElement } from 'cap-widget'

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

const containerRef = ref<HTMLElement | null>(null)
let widgetElement: CapWidgetElement | null = null
let creditsObserver: MutationObserver | null = null
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

const destroyWidget = () => {
  creditsObserver?.disconnect()
  creditsObserver = null
  if (!widgetElement) return
  widgetElement.removeEventListener('solve', handleSolve)
  widgetElement.removeEventListener('error', handleError)
  widgetElement.removeEventListener('reset', handleReset)
  widgetElement.remove()
  widgetElement = null
}

const hideWidgetCredits = (el: CapWidgetElement) => {
  const shadowRoot = el.shadowRoot
  if (!shadowRoot) return

  const removeCredits = () => {
    shadowRoot.querySelector('.credits')?.remove()
  }

  removeCredits()
  creditsObserver = new MutationObserver(removeCredits)
  creditsObserver.observe(shadowRoot, { childList: true, subtree: true })
}

const renderWidget = () => {
  destroyWidget()
  cachedToken = null
  if (!containerRef.value || !resolvedEndpoint.value) {
    return
  }

  const el = document.createElement('cap-widget') as CapWidgetElement
  el.setAttribute('data-cap-api-endpoint', resolvedEndpoint.value)
  // cap-widget defaults its internal panel to 260px. Override the custom
  // property on the host so it follows the full width of the auth form.
  el.style.setProperty('--cap-widget-width', '100%')
  if (props.theme) {
    el.setAttribute('data-cap-theme', props.theme)
  }

  el.addEventListener('solve', handleSolve)
  el.addEventListener('error', handleError)
  el.addEventListener('reset', handleReset)

  widgetElement = el
  containerRef.value.appendChild(el)
  hideWidgetCredits(el)
}

const reset = () => {
  cachedToken = null
  if (widgetElement) {
    try {
      widgetElement.reset()
    } catch {
      // Ignore reset failures
    }
  }
}

const verify = async (): Promise<string | null> => {
  if (cachedToken) {
    return cachedToken
  }
  if (widgetElement) {
    try {
      const res = await widgetElement.solve()
      if (res?.token) {
        cachedToken = res.token
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

watch(
  () => [props.apiEndpoint, props.siteKey],
  () => {
    nextTick(() => {
      renderWidget()
    })
  }
)

onMounted(() => {
  nextTick(() => {
    renderWidget()
  })
})

onUnmounted(() => {
  cachedToken = null
  destroyWidget()
})
</script>

<style scoped>
.cap-widget-wrapper {
  width: 100%;
  min-height: 65px;
  margin: 0.5rem 0;
  display: flex;
  justify-content: center;
}

.cap-widget-container {
  width: 100%;
  display: flex;
  justify-content: center;
}

.cap-widget-container :deep(cap-widget) {
  display: block;
  width: 100%;
  max-width: 100%;
}
</style>
