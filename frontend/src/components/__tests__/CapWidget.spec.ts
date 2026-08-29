import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: {
      endpoint: 'https://cap.example.com/site-key/',
      theme: 'auto'
    }
  })
}))

vi.mock('cap-widget', () => {
  class MockCapWidget extends HTMLElement {
    constructor() {
      super()
      const shadowRoot = this.attachShadow({ mode: 'open' })
      const panel = document.createElement('div')
      panel.className = 'captcha'
      const credits = document.createElement('a')
      credits.className = 'credits'
      credits.textContent = 'Cap'
      panel.appendChild(credits)
      shadowRoot.appendChild(panel)
    }

    async solve() {
      return { token: 'test-token' }
    }

    reset() {}
  }

  if (!customElements.get('cap-widget')) {
    customElements.define('cap-widget', MockCapWidget)
  }

  return { CapWidget: MockCapWidget }
})

import CapWidget from '@/components/CapWidget.vue'
import CapFrameView from '@/views/auth/CapFrameView.vue'

describe('CapWidget', () => {
  it('renders the widget in the isolated Cap frame', async () => {
    const wrapper = mount(CapWidget, {
      props: {
        apiEndpoint: 'https://cap.example.com',
        siteKey: 'site-key'
      }
    })

    await nextTick()
    await nextTick()

    const frame = wrapper.element.querySelector('iframe') as HTMLIFrameElement
    expect(frame).not.toBeNull()
    const src = new URL(frame.src)
    expect(src.pathname).toBe('/cap-frame')
    expect(src.searchParams.get('endpoint')).toBe('https://cap.example.com/site-key/')
    expect(src.searchParams.get('theme')).toBe('auto')
    wrapper.unmount()
  })

  it('fills the frame and removes Cap branding from the shadow root', async () => {
    const wrapper = mount(CapFrameView)

    await nextTick()
    await nextTick()

    const widget = wrapper.element.querySelector('cap-widget') as HTMLElement
    expect(widget.style.getPropertyValue('--cap-widget-width')).toBe('100%')
    expect(widget.shadowRoot?.querySelector('.credits')).toBeNull()

    const restoredCredits = document.createElement('a')
    restoredCredits.className = 'credits'
    restoredCredits.textContent = 'Cap'
    widget.shadowRoot?.querySelector('.captcha')?.appendChild(restoredCredits)
    await nextTick()

    expect(widget.shadowRoot?.querySelector('.credits')).toBeNull()
    wrapper.unmount()
  })
})
