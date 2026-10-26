<script setup lang="ts">
import { LoaderCircleIcon, RadioIcon, VolumeXIcon } from '@lucide/vue'

type Mode = 'hls' | 'webrtc'
type State = 'offline' | 'connecting' | 'playing' | 'error'

const props = defineProps<{
  hlsUrl: string
  whepUrl: string
  live: boolean
  /** Embed mode: no mode switcher below the video. */
  compact?: boolean
}>()

const MODE_KEY = 'playback-mode'
const SOUND_KEY = 'player-sound'

const video = ref<HTMLVideoElement | null>(null)
const mode = ref<Mode>('hls')
const state = ref<State>(props.live ? 'connecting' : 'offline')
const muted = ref(true)
const hint = ref('')
// The muted value we set ourselves, to tell our changes from the viewer's.
let expectedMuted = true

let current: { stop: () => void } | null = null
let starting = false
let retryTimer: ReturnType<typeof setTimeout> | undefined

const overlayText = computed(() => ({
  offline: 'The stream is offline. It starts here automatically when it goes live.',
  connecting: 'Connecting…',
  error: 'Couldn\'t play the stream in this browser.',
  playing: '',
})[state.value])

function soundPreferred() {
  try {
    return localStorage.getItem(SOUND_KEY) !== 'off'
  }
  catch {
    return true
  }
}

function setMuted(v: HTMLVideoElement, value: boolean) {
  expectedMuted = value
  v.muted = value
}

// Play with sound when the browser allows it and the viewer hasn't muted
// before. If the browser blocks sound (no click on this site yet), start
// muted and show "Tap to unmute" instead.
async function playBest(v: HTMLVideoElement) {
  setMuted(v, !soundPreferred())
  try {
    await v.play()
  }
  catch (err) {
    if (v.muted || (err as DOMException).name !== 'NotAllowedError') return
    setMuted(v, true)
    v.play().catch(() => {})
  }
  finally {
    muted.value = v.muted
  }
}

function onVolumeChange(event: Event) {
  const v = event.target as HTMLVideoElement
  muted.value = v.muted
  // Remember the viewer's own mute/unmute (not ours) for next time.
  if (v.muted !== expectedMuted) {
    expectedMuted = v.muted
    try {
      localStorage.setItem(SOUND_KEY, v.muted ? 'off' : 'on')
    }
    catch {}
  }
}

function resetVideo() {
  const v = video.value
  if (!v) return
  v.srcObject = null
  v.removeAttribute('src')
  v.load()
}

function stop() {
  current?.stop()
  current = null
}

// After a playback failure, try again while the channel is still live.
function retrySoon() {
  clearTimeout(retryTimer)
  retryTimer = setTimeout(() => {
    if (props.live && !current) start()
  }, 3000)
}

async function playHLS(url: string) {
  const v = video.value!
  const { default: Hls } = await import('hls.js')
  if (Hls.isSupported()) {
    const hls = new Hls({ lowLatencyMode: true, backBufferLength: 30 })
    hls.on(Hls.Events.ERROR, (_event, data) => {
      if (data.fatal) {
        console.warn('HLS error', data.details)
        stop()
        retrySoon()
      }
    })
    hls.loadSource(url)
    hls.attachMedia(v)
    playBest(v)
    return { stop: () => { hls.destroy(); resetVideo() } }
  }
  // Safari plays HLS natively.
  if (v.canPlayType('application/vnd.apple.mpegurl')) {
    v.src = url
    playBest(v)
    return { stop: resetVideo }
  }
  throw new Error('HLS is not supported in this browser')
}

function waitFor(pc: RTCPeerConnection, event: string, done: () => boolean, timeoutMs: number) {
  return new Promise<void>((resolve, reject) => {
    if (done()) return resolve()
    const check = () => {
      if (pc.connectionState === 'failed') finish(new Error('connection failed'))
      else if (done()) finish()
    }
    const timer = setTimeout(() => finish(new Error(`${event} timed out`)), timeoutMs)
    const finish = (err?: Error) => {
      clearTimeout(timer)
      pc.removeEventListener(event, check)
      if (err) reject(err)
      else resolve()
    }
    pc.addEventListener(event, check)
  })
}

// WebRTC playback via WHEP (RFC 9725).
async function playWebRTC(url: string) {
  const v = video.value!
  const pc = new RTCPeerConnection()
  const stream = new MediaStream()
  pc.addTransceiver('video', { direction: 'recvonly' })
  pc.addTransceiver('audio', { direction: 'recvonly' })
  pc.addEventListener('track', e => stream.addTrack(e.track))

  let session: string | null = null
  const stopRtc = () => {
    pc.close()
    if (session) fetch(session, { method: 'DELETE' }).catch(() => {})
    resetVideo()
  }

  try {
    await pc.setLocalDescription(await pc.createOffer())
    await waitFor(pc, 'icegatheringstatechange', () => pc.iceGatheringState === 'complete', 2000).catch(() => {})
    const endpoint = new URL(url, window.location.href)
    const res = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/sdp' },
      body: pc.localDescription!.sdp,
    })
    if (res.status !== 201) throw new Error(`WHEP request failed (${res.status})`)
    const location = res.headers.get('Location')
    if (location) session = new URL(location, endpoint).href
    await pc.setRemoteDescription({ type: 'answer', sdp: await res.text() })
    await waitFor(pc, 'connectionstatechange', () => pc.connectionState === 'connected', 10_000)
  }
  catch (err) {
    stopRtc()
    throw err
  }

  pc.addEventListener('connectionstatechange', () => {
    if (pc.connectionState === 'failed') {
      stop()
      retrySoon()
    }
  })
  v.srcObject = stream
  playBest(v)

  // WebRTC can't carry AAC, which OBS sends over RTMP, so check that
  // audio actually arrives.
  setTimeout(async () => {
    if (pc.connectionState !== 'connected') return
    const stats = [...(await pc.getStats()).values()]
    const audio = stats.find(s => s.type === 'inbound-rtp' && s.kind === 'audio')
    if (!audio?.packetsReceived) {
      hint.value = 'No sound in Low latency mode for this stream. Switch to Standard for sound.'
    }
  }, 3000)

  return { stop: stopRtc }
}

async function start() {
  if (starting || !video.value) return
  starting = true
  stop()
  hint.value = ''
  state.value = 'connecting'
  try {
    current = mode.value === 'webrtc' ? await playWebRTC(props.whepUrl) : await playHLS(props.hlsUrl)
  }
  catch (err) {
    console.warn(err)
    if (mode.value === 'webrtc') {
      hint.value = 'Low latency mode couldn\'t connect, using Standard instead.'
      try {
        current = await playHLS(props.hlsUrl)
      }
      catch (e) {
        console.warn(e)
      }
    }
    if (!current) state.value = 'error'
  }
  finally {
    starting = false
  }
  // The stream may have ended while we were connecting.
  if (!props.live) {
    stop()
    state.value = 'offline'
  }
}

function setMode(next: Mode) {
  if (next === mode.value) return
  mode.value = next
  try {
    localStorage.setItem(MODE_KEY, next)
  }
  catch {}
  if (props.live) start()
}

function unmute() {
  const v = video.value
  if (!v) return
  v.muted = false // a click: always allowed, and remembered by onVolumeChange
  v.play().catch(() => {})
}

watch(() => props.live, (live) => {
  if (live) {
    if (!current) start()
  }
  else {
    clearTimeout(retryTimer)
    stop()
    hint.value = ''
    state.value = 'offline'
  }
})

onMounted(() => {
  try {
    if (localStorage.getItem(MODE_KEY) === 'webrtc') mode.value = 'webrtc'
  }
  catch {}
  if (props.live) start()
})

onBeforeUnmount(() => {
  clearTimeout(retryTimer)
  stop()
})
</script>

<template>
  <div>
    <div class="relative aspect-video overflow-hidden rounded-xl bg-black" :class="{ 'rounded-none': compact }">
      <!-- Sound is decided in playBest(): on if the browser allows it. -->
      <video
        ref="video"
        class="size-full"
        playsinline
        controls
        @playing="state = 'playing'"
        @volumechange="onVolumeChange"
      />
      <div
        v-if="state !== 'playing'"
        class="absolute inset-0 grid place-items-center bg-card p-6 text-center text-sm text-muted-foreground"
      >
        <div class="flex max-w-sm flex-col items-center gap-3">
          <LoaderCircleIcon v-if="state === 'connecting'" class="size-6 animate-spin" />
          <RadioIcon v-else class="size-6" />
          <p>{{ overlayText }}</p>
        </div>
      </div>
      <button
        v-if="state === 'playing' && muted"
        type="button"
        class="absolute top-3 left-3 inline-flex items-center gap-1.5 rounded-md bg-black/70 px-3 py-2 text-sm font-semibold text-white hover:bg-black/80"
        @click="unmute"
      >
        <VolumeXIcon class="size-4" />
        Tap to unmute
      </button>
    </div>

    <div v-if="!compact" class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2">
      <div class="inline-flex rounded-lg border bg-card p-0.5 text-sm" role="radiogroup" aria-label="Playback mode">
        <button
          v-for="option in ([['hls', 'Standard'], ['webrtc', 'Low latency']] as const)"
          :key="option[0]"
          type="button"
          role="radio"
          :aria-checked="mode === option[0]"
          class="rounded-md px-3 py-1 transition-colors"
          :class="mode === option[0] ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'"
          @click="setMode(option[0])"
        >
          {{ option[1] }}
        </button>
      </div>
      <p v-if="hint" class="text-sm text-muted-foreground">
        {{ hint }}
      </p>
    </div>
  </div>
</template>
