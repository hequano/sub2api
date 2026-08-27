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
import { computed, onUnmounted, ref } from 'vue'
import 'cap-widget'
import Cap, { type CapWidget as CapWidgetElement } from 'cap-widget'

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

const widgetRef = ref<CapWidgetElement | null>(null)
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

const reset = () => {
  cachedToken = null
  if (widgetRef.value) {
    try {
      widgetRef.value.reset()
    } catch {
      // Ignore reset failures
    }
  }
}

const verify = async (): Promise<string | null> => {
  if (cachedToken) {
    return cachedToken
  }
  if (resolvedEndpoint.value) {
    try {
      const solver = new Cap({ apiEndpoint: resolvedEndpoint.value })
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
