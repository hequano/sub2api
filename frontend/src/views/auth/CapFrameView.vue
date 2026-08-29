<template>
  <main class="cap-frame" aria-label="Cap verification">
    <div ref="containerRef" class="cap-frame-container"></div>
  </main>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import 'cap-widget'
import { type CapWidget as CapWidgetElement } from 'cap-widget'

const MESSAGE_SOURCE = 'sub2api-cap-frame'
const route = useRoute()
const containerRef = ref<HTMLElement | null>(null)

let widgetElement: CapWidgetElement | null = null
let creditsObserver: MutationObserver | null = null
let resizeObserver: ResizeObserver | null = null

function queryString(name: string): string {
  const value = route.query[name]
  return typeof value === 'string' ? value : ''
}

function resolveEndpoint(): string {
  const endpoint = queryString('endpoint').trim()
  if (!endpoint) return ''
  try {
    const parsed = new URL(endpoint)
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return ''
    return parsed.toString()
  } catch {
    return ''
  }
}

function postToParent(payload: Record<string, unknown>) {
  if (window.parent === window) return
  window.parent.postMessage({ source: MESSAGE_SOURCE, ...payload }, window.location.origin)
}

function removeCredits(el: CapWidgetElement) {
  const shadowRoot = el.shadowRoot
  if (!shadowRoot) return

  const remove = () => shadowRoot.querySelector('.credits')?.remove()
  remove()
  creditsObserver = new MutationObserver(remove)
  creditsObserver.observe(shadowRoot, { childList: true, subtree: true })
}

function reportHeight() {
  const height = Math.ceil(widgetElement?.getBoundingClientRect().height || 65)
  postToParent({ event: 'resize', height })
}

function handleSolve(event: Event) {
  const token = (event as CustomEvent<{ token?: string }>).detail?.token || ''
  if (token) postToParent({ event: 'solve', token })
}

function handleError(event: Event) {
  const message = (event as CustomEvent<{ message?: string }>).detail?.message || 'Cap verification failed'
  postToParent({ event: 'error', message })
}

function handleReset() {
  postToParent({ event: 'reset' })
}

async function handleParentMessage(event: MessageEvent) {
  if (event.source !== window.parent || event.origin !== window.location.origin) return
  const data = event.data as { source?: string; action?: string; requestId?: string }
  if (data?.source !== MESSAGE_SOURCE || !widgetElement) return

  if (data.action === 'reset') {
    widgetElement.reset()
    return
  }
  if (data.action !== 'solve' || !data.requestId) return

  try {
    const result = await widgetElement.solve()
    postToParent({ event: 'result', requestId: data.requestId, token: result?.token || null })
  } catch (error) {
    postToParent({
      event: 'result',
      requestId: data.requestId,
      token: null,
      message: error instanceof Error ? error.message : 'Cap verification failed'
    })
  }
}

onMounted(() => {
  const endpoint = resolveEndpoint()
  if (!endpoint || !containerRef.value) {
    postToParent({ event: 'error', message: 'Invalid Cap endpoint' })
    return
  }

  const el = document.createElement('cap-widget') as CapWidgetElement
  el.setAttribute('data-cap-api-endpoint', endpoint)
  el.style.setProperty('--cap-widget-width', '100%')

  const theme = queryString('theme')
  if (theme === 'light' || theme === 'dark' || theme === 'auto') {
    el.setAttribute('data-cap-theme', theme)
  }

  el.addEventListener('solve', handleSolve)
  el.addEventListener('error', handleError)
  el.addEventListener('reset', handleReset)
  widgetElement = el
  containerRef.value.appendChild(el)

  removeCredits(el)
  resizeObserver = new ResizeObserver(reportHeight)
  resizeObserver.observe(el)
  requestAnimationFrame(reportHeight)
  window.addEventListener('message', handleParentMessage)
  postToParent({ event: 'ready' })
})

onUnmounted(() => {
  window.removeEventListener('message', handleParentMessage)
  resizeObserver?.disconnect()
  creditsObserver?.disconnect()
  if (widgetElement) {
    widgetElement.removeEventListener('solve', handleSolve)
    widgetElement.removeEventListener('error', handleError)
    widgetElement.removeEventListener('reset', handleReset)
    widgetElement.remove()
  }
  widgetElement = null
})
</script>

<style scoped>
:global(html),
:global(body),
:global(#app) {
  width: 100%;
  min-height: 0;
  margin: 0;
  padding: 0;
  overflow: hidden;
  background: transparent;
}

.cap-frame,
.cap-frame-container,
.cap-frame-container :deep(cap-widget) {
  display: block;
  width: 100%;
  max-width: 100%;
  margin: 0;
  padding: 0;
}
</style>
