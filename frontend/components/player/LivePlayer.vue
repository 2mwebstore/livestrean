<script setup lang="ts">
import { LoaderCircleIcon, MaximizeIcon, MinimizeIcon, RadioIcon, Volume2Icon, VolumeXIcon } from '@lucide/vue'

type Mode = 'hls' | 'webrtc'
type State = 'offline' | 'connecting' | 'playing' | 'error'

const props = defineProps<{
  hlsUrl: string
  whepUrl: string
  live: boolean
  /** Embed mode: no mode switcher below the video. */
  compact?: boolean
  /** Image shown behind the offline message (the channel thumbnail). */
  poster?: string | null
  /** Shown next to the LIVE label on the video (used by the embed). */
  viewers?: number | null
}>()
// #offline: replaces the offline message, e.g. with a countdown.
defineSlots<{ offline?: () => unknown }>()

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
// muted; the LIVE label then shows a muted speaker and a tap turns sound on.
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

// --- Controls: tap the video to mute/unmute; LIVE label when idle ----------

const container = ref<HTMLElement | null>(null)
const hovering = ref(false) // mouse over the video
const tapped = ref(false) // recently clicked/tapped
const fullscreen = ref(false)
// Brief icon in the middle of the video after a tap.
const feedback = ref<'muted' | 'unmuted' | null>(null)
let tapTimer: ReturnType<typeof setTimeout> | undefined
let feedbackTimer: ReturnType<typeof setTimeout> | undefined

// While the viewer interacts, the LIVE label hides and the fullscreen
// button shows; otherwise the other way round.
const interacting = computed(() => hovering.value || tapped.value)

function onPointerEnter(event: PointerEvent) {
  if (event.pointerType === 'mouse') hovering.value = true
}

function onPointerLeave() {
  hovering.value = false
}

function markTapped() {
  tapped.value = true
  clearTimeout(tapTimer)
  tapTimer = setTimeout(() => (tapped.value = false), 2500)
}

// Runs inside the click, so browsers allow turning the sound on.
function toggleMute() {
  const v = video.value
  if (!v) return
  markTapped()
  v.muted = !v.muted // remembered by onVolumeChange
  if (!v.muted) v.play().catch(() => {})
  feedback.value = v.muted ? 'muted' : 'unmuted'
  clearTimeout(feedbackTimer)
  feedbackTimer = setTimeout(() => (feedback.value = null), 700)
}

function toggleFullscreen() {
  markTapped()
  const v = video.value as (HTMLVideoElement & { webkitEnterFullscreen?: () => void }) | null
  if (document.fullscreenElement) {
    document.exitFullscreen().catch(() => {})
  }
  else if (container.value?.requestFullscreen) {
    container.value.requestFullscreen().catch(() => {})
  }
  else {
    v?.webkitEnterFullscreen?.() // iPhone: only the video itself can go fullscreen
  }
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'f') toggleFullscreen()
  else if (event.key === 'm') toggleMute()
}

function onFullscreenChange() {
  fullscreen.value = document.fullscreenElement === container.value
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
  document.addEventListener('fullscreenchange', onFullscreenChange)
  try {
    if (localStorage.getItem(MODE_KEY) === 'webrtc') mode.value = 'webrtc'
  }
  catch {}
  if (props.live) start()
})

onBeforeUnmount(() => {
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  clearTimeout(tapTimer)
  clearTimeout(feedbackTimer)
  clearTimeout(retryTimer)
  stop()
})
</script>

<template>
  <div>
    <div
      ref="container"
      class="relative aspect-video overflow-hidden rounded-xl bg-black select-none"
      :class="{ 'rounded-none': compact || fullscreen }"
      @pointerenter="onPointerEnter"
      @pointerleave="onPointerLeave"
    >
      <!-- No browser controls: a tap on the video mutes/unmutes instead.
           Sound is decided in playBest(): on if the browser allows it. -->
      <video
        ref="video"
        class="size-full"
        playsinline
        @playing="state = 'playing'"
        @volumechange="onVolumeChange"
      />
      <div
        v-if="state !== 'playing'"
        class="absolute inset-0 grid place-items-center overflow-hidden bg-card p-6 text-center text-sm text-muted-foreground"
      >
        <img
          v-if="poster && state === 'offline'"
          :src="poster"
          alt=""
          class="absolute inset-0 size-full object-cover opacity-35"
        >
        <div class="relative flex max-w-md flex-col items-center gap-3" :class="{ 'text-white': poster && state === 'offline' }">
          <slot v-if="state === 'offline' && $slots.offline" name="offline" />
          <template v-else>
            <LoaderCircleIcon v-if="state === 'connecting'" class="size-6 animate-spin" />
            <RadioIcon v-else class="size-6" />
            <p>{{ overlayText }}</p>
          </template>
        </div>
      </div>
      <template v-if="state === 'playing'">
        <!-- The whole video is the mute/unmute button. -->
        <button
          type="button"
          class="absolute inset-0 cursor-pointer focus-visible:ring-2 focus-visible:ring-white/70 focus-visible:outline-none focus-visible:ring-inset"
          :aria-label="muted ? 'Unmute (M). Fullscreen: F' : 'Mute (M). Fullscreen: F'"
          @click="toggleMute"
          @keydown="onKeydown"
        />

        <!-- LIVE label: shown while idle, hidden while hovering or just after a tap. -->
        <div
          class="pointer-events-none absolute top-3 left-3 flex items-center gap-1.5 transition-opacity duration-200"
          :class="interacting ? 'opacity-0' : 'opacity-100'"
          aria-hidden="true"
        >
          <span class="inline-flex items-center gap-1.5 rounded-md bg-live px-2 py-1 text-xs font-bold tracking-wider text-white uppercase shadow">
            <span class="size-1.5 animate-pulse rounded-full bg-white" />
            Live
            <VolumeXIcon v-if="muted" class="size-3.5" />
          </span>
          <span v-if="viewers != null" class="rounded-md bg-black/70 px-1.5 py-0.5 text-xs text-white">
            <ViewerCount :count="viewers" />
          </span>
        </div>

        <!-- Brief confirmation in the middle after a tap. -->
        <div
          class="pointer-events-none absolute inset-0 grid place-items-center transition-opacity duration-300"
          :class="feedback ? 'opacity-100' : 'opacity-0'"
          aria-hidden="true"
        >
          <span class="grid size-16 place-items-center rounded-full bg-black/60 text-white">
            <VolumeXIcon v-if="feedback === 'muted'" class="size-7" />
            <Volume2Icon v-else class="size-7" />
          </span>
        </div>

        <button
          type="button"
          class="absolute right-3 bottom-3 grid size-9 place-items-center rounded-md bg-black/60 text-white transition-opacity duration-200 hover:bg-black/80 focus-visible:opacity-100"
          :class="interacting ? 'opacity-100' : 'pointer-events-none opacity-0'"
          :aria-label="fullscreen ? 'Exit fullscreen' : 'Fullscreen'"
          @click.stop="toggleFullscreen"
        >
          <MinimizeIcon v-if="fullscreen" class="size-4" />
          <MaximizeIcon v-else class="size-4" />
        </button>
      </template>
      <!-- Mute state for screen readers -->
      <span class="sr-only" aria-live="polite">{{ state === 'playing' ? (muted ? 'Sound off' : 'Sound on') : '' }}</span>
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
