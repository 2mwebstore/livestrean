// End-to-end test against the running Docker stack, in a real browser.
//
//   docker compose up -d --build        (from the project root)
//   node frontend/tests/e2e/full-flow.mjs
//
// Env: BASE_URL (default http://localhost:8080), CHROME_PATH, ADMIN_EMAIL /
// ADMIN_PASSWORD (default: read from ../.env), SCREENSHOTS (directory, optional).
import { spawn } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import puppeteer from 'puppeteer-core'

const root = join(dirname(fileURLToPath(import.meta.url)), '../../..')
const env = Object.fromEntries(
  readFileSync(join(root, '.env'), 'utf8').split('\n')
    .filter(l => /^[A-Z_]+=/.test(l))
    .map(l => [l.slice(0, l.indexOf('=')), l.slice(l.indexOf('=') + 1).replace(/^"|"$/g, '')]),
)
const BASE = process.env.BASE_URL ?? 'http://localhost:8080'
const ADMIN_EMAIL = process.env.ADMIN_EMAIL ?? env.ADMIN_EMAIL
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD ?? env.ADMIN_PASSWORD
const CHROME = process.env.CHROME_PATH ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
const SHOTS = process.env.SCREENSHOTS

const run = Date.now().toString(36)
const results = []
function check(name, ok, detail = '') {
  results.push({ name, ok })
  console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}${detail ? `  (${detail})` : ''}`)
}
const sleep = ms => new Promise(r => setTimeout(r, ms))
async function shot(page, name) {
  if (SHOTS) await page.screenshot({ path: join(SHOTS, `${name}.png`) })
}

// Admin API session, used only to promote the streamer.
async function adminPromote(userId) {
  const login = await fetch(`${BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD }),
  })
  const cookie = login.headers.get('set-cookie').split(';')[0]
  const res = await fetch(`${BASE}/api/v1/admin/users/${userId}/role`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', cookie },
    body: JSON.stringify({ role: 'streamer' }),
  })
  return res.status
}

async function signUp(page, name) {
  await page.goto(`${BASE}/register`, { waitUntil: 'networkidle0' })
  await page.type('#name', name)
  await page.type('#email', `${name.toLowerCase().replace(/\W/g, '')}-${run}@e2e.test`)
  await page.type('#password', 'password-123')
  await Promise.all([page.waitForNavigation({ waitUntil: 'networkidle0' }), page.click('button[type=submit]')])
  return page.evaluate(async () => (await (await fetch('/api/v1/auth/me')).json()).data)
}

async function videoPlaying(page, timeout = 30_000) {
  try {
    await page.waitForFunction(() => {
      const v = document.querySelector('video')
      return v && v.readyState >= 3 && v.currentTime > 0.5 && v.videoWidth > 0
    }, { timeout })
    return true
  }
  catch {
    return false
  }
}

const text = (page, selector) => page.$eval(selector, el => el.textContent.trim()).catch(() => '')

const browser = await puppeteer.launch({ executablePath: CHROME, headless: true })
let stream
try {
  // 1. Streamer signs up (as a viewer), the admin promotes them.
  const streamerCtx = await browser.createBrowserContext()
  const sp = await streamerCtx.newPage()
  await sp.setViewport({ width: 1280, height: 860 })
  const streamer = await signUp(sp, 'Streamer E2E')
  check('sign up through the UI', streamer?.role === 'viewer', streamer?.role)
  check('admin promotes user to streamer', await adminPromote(streamer.id) === 200)

  // 2. Create a channel from the dashboard and read the key from the dialog.
  await sp.goto(`${BASE}/dashboard`, { waitUntil: 'networkidle0' })
  check('dashboard opens for streamer', (await text(sp, 'h1')) === 'Your channels')
  await sp.locator('button ::-p-text(New channel)').click()
  await sp.waitForSelector('#channel-title')
  await sp.type('#channel-title', `E2E Show ${run}`)
  await sp.click('[role=dialog] button[type=submit]')
  await sp.waitForFunction(() => document.querySelector('[role=dialog]')?.textContent.includes('Copy it now'), { timeout: 10_000 })
  const fields = await sp.$$eval('[role=dialog] input', inputs => inputs.map(i => i.value))
  const [rtmpServer, rtmpKey, whipUrl] = fields
  const slug = rtmpKey.split('?')[0]
  const key = new URLSearchParams(rtmpKey.split('?')[1]).get('pass')
  check('stream key shown once after creating', rtmpServer.endsWith('/live') && key.startsWith('sk_') && whipUrl.endsWith(`/live/${slug}/whip`), `${rtmpServer} · ${slug}`)
  await shot(sp, '1-stream-key-dialog')
  await sp.locator('button ::-p-text(I\'ve copied it)').click()

  // 2b. Thumbnail and countdown, set from the channel's dashboard page.
  const channelId = await sp.evaluate(async s => (await (await fetch('/api/v1/me/channels')).json()).data.find(c => c.slug === s).id, slug)
  await sp.goto(`${BASE}/dashboard/channels/${channelId}`, { waitUntil: 'networkidle0' })
  const imagePath = join(tmpdir(), `e2e-thumbnail-${run}.png`)
  await sp.screenshot({ path: imagePath }) // any PNG will do
  const [chooser] = await Promise.all([sp.waitForFileChooser(), sp.locator('button ::-p-text(Upload image)').click()])
  await chooser.accept([imagePath])
  const thumbOk = await sp.waitForSelector('img[alt="Channel thumbnail"]', { timeout: 15_000 }).then(() => true, () => false)
  check('upload a thumbnail from the dashboard', thumbOk)

  const inHalfHour = new Date(Date.now() + 30 * 60_000)
  const pad = n => String(n).padStart(2, '0')
  const local = `${inHalfHour.getFullYear()}-${pad(inHalfHour.getMonth() + 1)}-${pad(inHalfHour.getDate())}T${pad(inHalfHour.getHours())}:${pad(inHalfHour.getMinutes())}`
  await sp.$eval('input[type="datetime-local"]', (el, v) => {
    el.value = v
    el.dispatchEvent(new Event('input', { bubbles: true }))
  }, local)
  await sp.locator('form button[type=submit] ::-p-text(Save)').click()
  const scheduled = await sp.waitForFunction(() => document.body.textContent.includes('Scheduled for'), { timeout: 10_000 }).then(() => true, () => false)
  check('schedule the next stream from the dashboard', scheduled)
  await shot(sp, '0-dashboard-thumbnail-schedule')

  const early = await (await browser.createBrowserContext()).newPage()
  await early.setViewport({ width: 1280, height: 860 })
  await early.goto(`${BASE}/c/${slug}`, { waitUntil: 'networkidle0' })
  const countdown = await early.waitForSelector('[role=timer]', { timeout: 10_000 }).then(() => true, () => false)
  check('visitor sees a countdown before the stream', countdown, await early.$eval('[role=timer]', el => el.getAttribute('aria-label')).catch(() => ''))
  check('thumbnail shown behind the player while offline', await early.$$eval('img', imgs => imgs.some(i => i.src.includes('/thumbnail?v='))))
  await shot(early, '0-channel-countdown')
  await early.goto(`${BASE}/`, { waitUntil: 'networkidle0' })
  const card = await early.$eval(`a[href="/c/${slug}"]`, a => ({ thumb: !!a.querySelector('img[src*="/thumbnail?v="]'), text: a.textContent }))
  check('home page card shows the thumbnail and start time', card.thumb && card.text.includes('Starts in'), card.text.trim().replace(/\s+/g, ' '))

  // 3. Stream to it exactly like OBS would.
  stream = spawn(join(root, 'scripts/test-stream.sh'), [slug, key], { stdio: 'ignore' })
  await early.goto(`${BASE}/c/${slug}`, { waitUntil: 'load' })
  const replaced = await videoPlaying(early, 40_000) && !(await early.$('[role=timer]'))
  const cleared = await early.evaluate(async s => (await (await fetch(`/api/v1/channels/${s}`)).json()).data.scheduled_at, slug)
  check('going live replaces the countdown with video', replaced)
  check('going live clears the schedule', cleared === null, String(cleared))
  await early.close()

  // 4. A logged-in viewer and an anonymous visitor watch.
  const viewerCtx = await browser.createBrowserContext()
  const vp = await viewerCtx.newPage()
  await vp.setViewport({ width: 1280, height: 860 })
  await signUp(vp, 'Viewer E2E')
  await vp.goto(`${BASE}/c/${slug}`, { waitUntil: 'load' })
  check('viewer sees the stream play', await videoPlaying(vp, 40_000))
  check('LIVE badge shown', (await vp.$$eval('span', els => els.some(e => e.textContent.trim() === 'Live'))))

  const anonCtx = await browser.createBrowserContext()
  const ap = await anonCtx.newPage()
  await ap.setViewport({ width: 1280, height: 860 })
  await ap.goto(`${BASE}/c/${slug}`, { waitUntil: 'load' })
  check('anonymous visitor sees the stream play', await videoPlaying(ap))
  check('anonymous visitor gets "Log in to join the chat", no input',
    (await ap.content()).includes('Log in to join the chat') && !(await ap.$('input[aria-label="Chat message"]')))

  await vp.waitForFunction(() => document.querySelector('[title$="watching now"]')?.textContent.trim().startsWith('2'), { timeout: 8000 }).catch(() => {})
  check('viewer count updates live (2 watching)', (await text(vp, '[title$="watching now"]')).startsWith('2'), await text(vp, '[title$="watching now"]'))

  // 5. Chat: logged-in viewer posts; everyone sees it instantly.
  const message = `hello from e2e ${run}`
  await vp.type('input[aria-label="Chat message"]', message)
  await vp.click('button[aria-label="Send message"]')
  const seen = async page => page.waitForFunction(m => document.querySelector('[role=log]')?.textContent.includes(m), { timeout: 5000 }, message).then(() => true, () => false)
  check('viewer\'s message appears in their chat', await seen(vp))
  check('message reaches the anonymous visitor in real time', await seen(ap))
  const anonPost = await ap.evaluate(async s => (await fetch(`/api/v1/channels/${s}/comments`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ body: 'sneaky' }),
  })).status, slug)
  check('API refuses chat from a logged-out visitor', anonPost === 401, String(anonPost))
  await shot(vp, '2-channel-page-viewer')
  await shot(ap, '3-channel-page-anonymous')

  // 6. Home page lists the channel as live.
  const hp = await anonCtx.newPage()
  await hp.setViewport({ width: 1280, height: 860 })
  await hp.goto(`${BASE}/`, { waitUntil: 'networkidle0' })
  check('home page lists the channel under Live now', (await hp.content()).includes(`E2E Show ${run}`))
  await shot(hp, '4-home')

  // 7. Embed page plays too.
  const ep = await anonCtx.newPage()
  await ep.setViewport({ width: 960, height: 540 })
  await ep.goto(`${BASE}/embed/${slug}`, { waitUntil: 'load' })
  check('embed player plays', await videoPlaying(ep))
  await shot(ep, '5-embed')

  // 8. Streamer regenerates the key: the stream is cut off for viewers.
  const id = await sp.evaluate(async () => (await (await fetch('/api/v1/me/channels')).json()).data[0].id)
  await sp.goto(`${BASE}/dashboard/channels/${id}`, { waitUntil: 'load' })
  await sp.waitForSelector('::-p-text(New stream key)')
  await shot(sp, '6-dashboard-channel')
  await sp.locator('button ::-p-text(New stream key)').click()
  await sp.locator('[role=alertdialog] button ::-p-text(Generate new key)').click()
  await sp.waitForFunction(() => document.querySelector('[role=dialog]')?.textContent.includes('Copy it now'), { timeout: 10_000 })
  const offline = await vp.waitForFunction(() => document.body.textContent.includes('The stream is offline'), { timeout: 15_000 }).then(() => true, () => false)
  check('regenerating the key disconnects the stream', offline)
}
finally {
  stream?.kill('SIGINT')
  await browser.close()
}

const failed = results.filter(r => !r.ok).length
console.log(`\n${results.length - failed}/${results.length} checks passed`)
process.exit(failed ? 1 : 0)
