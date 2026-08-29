<template>
  <div v-if="siteKey && apiEndpoint" class="cap-widget-wrapper">
    <iframe
      ref="frameRef"
      class="cap-widget-frame"
      :src="frameSrc"
      :style="{ height: `${frameHeight}px` }"
      title="Cap verification"
      scrolling="no"
    ></iframe>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'

const MESSAGE_SOURCE = 'sub2api-cap-frame'
const VERIFY_TIMEOUT_MS = 60_000

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

const frameRef = ref<HTMLIFrameElement | null>(null)
const frameHeight = ref(65)
let cachedToken: string | null = null
let requestSequence = 0
let frameReady = false
const pendingVerifications = new Map<
  string,
  { resolve: (token: string | null) => void; timeout: number; sent: boolean }
>()

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

const frameSrc = computed(() => {
  const query = new URLSearchParams({
    endpoint: resolvedEndpoint.value,
    theme: props.theme
  })
  return `/cap-frame?${query.toString()}`
})

const postToFrame = (payload: Record<string, unknown>) => {
  frameRef.value?.contentWindow?.postMessage(
    { source: MESSAGE_SOURCE, ...payload },
    window.location.origin
  )
}

const settleVerification = (requestId: string, token: string | null) => {
  const pending = pendingVerifications.get(requestId)
  if (!pending) return
  window.clearTimeout(pending.timeout)
  pendingVerifications.delete(requestId)
  pending.resolve(token)
}

const handleFrameMessage = (event: MessageEvent) => {
  if (event.source !== frameRef.value?.contentWindow || event.origin !== window.location.origin) return
  const data = event.data as {
    source?: string
    event?: string
    token?: string | null
    requestId?: string
    message?: string
    height?: number
  }
  if (data?.source !== MESSAGE_SOURCE) return

  if (data.event === 'ready') {
    frameReady = true
    for (const [requestId, pending] of pendingVerifications) {
      if (!pending.sent) {
        pending.sent = true
        postToFrame({ action: 'solve', requestId })
      }
    }
    return
  }
  if (data.event === 'resize' && typeof data.height === 'number') {
    frameHeight.value = Math.min(110, Math.max(58, Math.ceil(data.height)))
    return
  }
  if (data.event === 'solve' && data.token) {
    cachedToken = data.token
    emit('verify', data.token)
    return
  }
  if (data.event === 'reset') {
    cachedToken = null
    emit('expire')
    return
  }
  if (data.event === 'error') {
    console.error('[Cap] Widget error:', data.message)
    cachedToken = null
    emit('error')
    return
  }
  if (data.event === 'result' && data.requestId) {
    const token = data.token || null
    if (token) cachedToken = token
    settleVerification(data.requestId, token)
  }
}

const reset = () => {
  cachedToken = null
  postToFrame({ action: 'reset' })
}

const verify = async (): Promise<string | null> => {
  if (cachedToken) {
    return cachedToken
  }
  if (!frameRef.value?.contentWindow) return null

  const requestId = `cap-${Date.now()}-${++requestSequence}`
  return new Promise<string | null>((resolve) => {
    const timeout = window.setTimeout(() => {
      pendingVerifications.delete(requestId)
      console.error('[Cap] Programmatic solve timed out')
      resolve(null)
    }, VERIFY_TIMEOUT_MS)
    pendingVerifications.set(requestId, { resolve, timeout, sent: frameReady })
    if (frameReady) postToFrame({ action: 'solve', requestId })
  })
}

const getToken = (): string | null => {
  return cachedToken
}

defineExpose({ verify, reset, getToken })

watch(
  () => [props.apiEndpoint, props.siteKey],
  () => {
    cachedToken = null
    frameReady = false
    frameHeight.value = 65
    for (const requestId of pendingVerifications.keys()) settleVerification(requestId, null)
  }
)

onMounted(() => {
  window.addEventListener('message', handleFrameMessage)
})

onUnmounted(() => {
  window.removeEventListener('message', handleFrameMessage)
  cachedToken = null
  for (const [requestId, pending] of pendingVerifications) {
    window.clearTimeout(pending.timeout)
    pendingVerifications.delete(requestId)
    pending.resolve(null)
  }
})
</script>

<style scoped>
.cap-widget-wrapper {
  width: 100%;
  min-height: 65px;
  margin: 0.5rem 0;
}

.cap-widget-frame {
  display: block;
  width: 100%;
  max-width: 100%;
  border: 0;
  overflow: hidden;
  background: transparent;
}
</style>
