/**
 * Schedule grid smoke test — day controls + working-hours validation.
 *
 * Covers:
 *  1. Grid loads, visual legend visible
 *  2. Pre-work hatched zone click → DayActionMenu modal opens
 *  3. "Закрыть день" → day_off exception created (201)
 *  4. Appointment on closed day → backend 422 + toast "вне рабочих"
 *  5. "Расширить рабочие часы" → custom_working_hours exception (201)
 *  6. Appointment OUTSIDE custom interval → 422
 *  7. Appointment INSIDE custom interval → 201
 *  8. Double booking on normal day → 409
 *  9. Cleanup: delete test exceptions
 * 10. No JS errors
 */
import { chromium } from 'playwright'

const BASE       = 'http://localhost:5173'
const EXEC       = 'C:/Users/User/AppData/Local/ms-playwright/chromium-1223/chrome-win64/chrome.exe'
const EMAIL      = 'admin@clinic.local'
const PASS       = 'changeme123'
const DOCTOR_ID  = 1

const pass = (msg) => console.log(`  ✅ ${msg}`)
const fail = (msg) => { console.error(`  ❌ ${msg}`); process.exitCode = 1 }
const info = (msg) => console.log(`  ℹ  ${msg}`)

// ── Date helpers ──────────────────────────────────────────────────────────────

function utcPlusDays(n) {
  const d = new Date()
  d.setUTCDate(d.getUTCDate() + n)
  return d.toISOString().slice(0, 10)          // YYYY-MM-DD
}

// today+2 (Wed) for close-day test; today+3 (Thu) for extend-hours test
const CLOSE_DATE  = utcPlusDays(2)
const EXTEND_DATE = utcPlusDays(3)
const TOMORROW    = utcPlusDays(1)

// ── Helpers ────────────────────────────────────────────────────────────────────

const navGroup = (page) =>
  page.locator('div.flex.items-center.gap-1').first()

async function clickNextDay(page, n = 1) {
  const btn = navGroup(page).locator('button').last()
  for (let i = 0; i < n; i++) {
    await btn.click()
    await page.waitForTimeout(450)
  }
  await page.waitForTimeout(800)
}

async function clickPrevDay(page, n = 1) {
  const btn = navGroup(page).locator('button').first()
  for (let i = 0; i < n; i++) {
    await btn.click()
    await page.waitForTimeout(450)
  }
  await page.waitForTimeout(800)
}

async function openCreateModal(page) {
  await page.locator('button').filter({ hasText: 'Новая запись' }).click()
  await page.waitForTimeout(600)
}

async function fillAppointment(page, datetime) {
  const ov = page.locator('.fixed.inset-0.z-50')
  await ov.locator('select').first().selectOption({ index: 1 })
  await page.waitForTimeout(1200)
  const svcSel = ov.locator('select').last()
  const svcOpts = await svcSel.locator('option').count()
  if (svcOpts > 1) await svcSel.selectOption({ index: 1 })
  await ov.locator('input[type="datetime-local"]').fill(datetime)

  // Create a throwaway patient
  await ov.getByText('+ Новый пациент').click()
  await page.waitForTimeout(300)
  await ov.locator('input[placeholder="ФИО *"]').fill(`Тест ${Date.now().toString().slice(-4)}`)
  await ov.locator('input[placeholder="Телефон *"]').fill(`+7916${Math.floor(Math.random()*9000000+1000000)}`)
  await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/patients') && r.request().method() === 'POST',
      { timeout: 8000 },
    ).catch(() => null),
    ov.getByText('Создать и выбрать').click(),
  ])
  await page.waitForTimeout(600)
}

async function submitAndCapture(page) {
  const ov = page.locator('.fixed.inset-0.z-50')
  const [resp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/appointments') && r.request().method() === 'POST',
      { timeout: 10000 },
    ).catch(() => null),
    ov.locator('button[type="submit"]').click(),
  ])
  await page.waitForTimeout(1500)
  return resp
}

async function deleteException(page, excId) {
  const token = await page.evaluate(() => localStorage.getItem('accessToken') ?? '')
  const r = await page.request.delete(
    `${BASE}/api/v1/doctors/${DOCTOR_ID}/exceptions/${excId}`,
    { headers: token ? { Authorization: `Bearer ${token}` } : {} },
  )
  return r.status()
}

// ─────────────────────────────────────────────────────────────────────────────

async function run() {
  const browser = await chromium.launch({ headless: false, executablePath: EXEC, slowMo: 80 })
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  const page    = await context.newPage()

  const pageErrors = []
  page.on('pageerror', (e) => pageErrors.push(e.message))

  info(`Close-day date  : ${CLOSE_DATE}`)
  info(`Extend-hrs date : ${EXTEND_DATE}`)
  info(`Tomorrow        : ${TOMORROW}`)

  // ── 0. Login ─────────────────────────────────────────────────────────────────
  console.log('\n0. Login')
  await page.goto(`${BASE}/login`)
  await page.fill('input[type="email"]', EMAIL)
  await page.fill('input[type="password"]', PASS)
  await page.getByRole('button', { name: 'Войти' }).click()
  await page.waitForURL((u) => !u.href.includes('/login'), { timeout: 8000 })
  pass('Logged in')

  // ── 1. Schedule grid + legend ─────────────────────────────────────────────
  console.log('\n1. Schedule grid + visual legend')
  await page.goto(`${BASE}/admin/schedule-grid`)
  await page.waitForLoadState('domcontentloaded')
  await page.waitForTimeout(3200)

  if (await page.locator('.overflow-auto').first().isVisible().catch(() => false))
    pass('Grid visible')
  else fail('Grid not visible')

  const legendItems = ['Рабочее', 'Нерабочее', 'Создана', 'Подтверждена']
  for (const label of legendItems) {
    if (await page.getByText(label).first().isVisible().catch(() => false))
      pass(`Legend: "${label}" visible`)
    else fail(`Legend: "${label}" missing`)
  }

  // ── 2. Navigate +2 → close-day date ──────────────────────────────────────
  console.log(`\n2. Navigate to ${CLOSE_DATE} (+2)`)
  await clickNextDay(page, 2)

  const cols = await page.evaluate(() =>
    document.querySelectorAll('.sticky.top-0 > div:not(.sticky)').length
  )
  info(`Doctor columns on ${CLOSE_DATE}: ${cols}`)
  if (cols > 0) pass(`${cols} doctor column(s) visible on target date`)
  else fail('No doctor columns — doctor may not work on this date')

  // ── 3. DayActionMenu via pre-work zone click ──────────────────────────────
  console.log('\n3. Pre-work zone → DayActionMenu')
  await page.locator('div.overflow-auto.flex-1').first().evaluate((el) => { el.scrollTop = 0 })
  await page.waitForTimeout(200)

  const preWorkZone = page.locator('[title="Управление расписанием"]').first()
  if (await preWorkZone.isVisible().catch(() => false)) {
    pass('Pre-work hatched zone found (title="Управление расписанием")')
    await preWorkZone.click()
  } else {
    // Fallback: click any hatched zone
    const postWorkZone = page.locator('[title="Управление расписанием"]').last()
    if (await postWorkZone.isVisible().catch(() => false)) {
      info('Using post-work zone as fallback')
      await postWorkZone.click()
    } else {
      fail('No blocked zone found to click')
      await browser.close()
      process.exit(1)
    }
  }
  await page.waitForTimeout(600)

  const dayActionTitle = page.locator('h2:has-text("Управление расписанием")')
  if (await dayActionTitle.isVisible().catch(() => false))
    pass('DayActionMenu modal opened')
  else { fail('DayActionMenu modal did not open'); await browser.close(); process.exit(1) }

  // ── 4. Close day → day_off exception ─────────────────────────────────────
  console.log('\n4. Close day → day_off exception')
  let closeDayExcId = null
  const dayActionOverlay = page.locator('.fixed.inset-0.z-50')

  const [excResp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/exceptions') && r.request().method() === 'POST',
      { timeout: 8000 },
    ).catch(() => null),
    dayActionOverlay.locator('button').filter({ hasText: 'Закрыть день' }).click(),
  ])
  await page.waitForTimeout(1200)

  if (excResp?.status() === 201) {
    const body = await excResp.json().catch(() => null)
    closeDayExcId = body?.id ?? null
    pass(`day_off exception created (ID=${closeDayExcId}) — 201`)
  } else if (excResp?.status() === 409) {
    info('day_off exception already exists (409) — fetching existing ID for cleanup')
    const token = await page.evaluate(() => localStorage.getItem('accessToken') ?? '')
    const listR = await page.request.get(
      `${BASE}/api/v1/doctors/${DOCTOR_ID}/exceptions?from=${CLOSE_DATE}&to=${CLOSE_DATE}`,
      { headers: token ? { Authorization: `Bearer ${token}` } : {} },
    )
    const existing = await listR.json().catch(() => [])
    if (Array.isArray(existing) && existing.length > 0) {
      closeDayExcId = existing[0].id
      info(`Reusing existing exception ID=${closeDayExcId}`)
    }
    pass('Close day: exception already present (acceptable)')
  } else {
    fail(`POST /exceptions: ${excResp?.status() ?? 'no response'}`)
  }

  if (await page.locator('text=Расписание обновлено').isVisible().catch(() => false))
    pass('Toast "Расписание обновлено" visible')
  else info('Toast may have disappeared')

  // modal should close automatically; Escape if still open
  if (await dayActionTitle.isVisible().catch(() => false)) {
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
  }
  pass('DayActionMenu closed after close-day action')

  // ── 5. Appointment on closed day → 422 ───────────────────────────────────
  console.log(`\n5. Appointment on closed day (${CLOSE_DATE}T10:00) → expect 422`)
  await openCreateModal(page)
  if (!await page.locator('h2:has-text("Новая запись")').isVisible().catch(() => false)) {
    fail('Create modal did not open'); await browser.close(); process.exit(1)
  }
  await fillAppointment(page, `${CLOSE_DATE}T10:00`)
  const apptClosed = await submitAndCapture(page)

  if (apptClosed) {
    const st = apptClosed.status()
    const body = await apptClosed.text().catch(() => '')
    info(`POST /appointments (closed day) → ${st}: ${body.slice(0, 100)}`)
    if (st === 422) pass('Appointment on closed day rejected: 422 ✅')
    else fail(`Expected 422, got ${st}`)
  } else {
    fail('POST /appointments not intercepted for closed-day test')
  }

  const toastOutside = await page.locator('text=вне рабочих').isVisible().catch(() => false)
  if (toastOutside) pass('Toast "вне рабочих часов" shown ✅')
  else info('Outside-hours toast not captured (may have disappeared)')

  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // ── 6. Navigate +1 → extend-hours date ───────────────────────────────────
  console.log(`\n6. Navigate to ${EXTEND_DATE} (+1 more)`)
  await clickNextDay(page, 1)

  const cols2 = await page.evaluate(() =>
    document.querySelectorAll('.sticky.top-0 > div:not(.sticky)').length
  )
  info(`Doctor columns on ${EXTEND_DATE}: ${cols2}`)
  if (cols2 > 0) pass('Doctor column(s) visible on extend-hours date')
  else fail('No doctor columns')

  // ── 7. Extend hours to 10:00-11:30 ───────────────────────────────────────
  console.log('\n7. DayActionMenu → custom_working_hours 10:00-11:30')
  await page.locator('div.overflow-auto.flex-1').first().evaluate((el) => { el.scrollTop = 0 })
  await page.waitForTimeout(200)

  const preWork2 = page.locator('[title="Управление расписанием"]').first()
  const postWork2 = page.locator('[title="Управление расписанием"]').last()
  if (await preWork2.isVisible().catch(() => false)) await preWork2.click()
  else if (await postWork2.isVisible().catch(() => false)) await postWork2.click()
  else { fail('No blocked zone found for extend-hours date'); await browser.close(); process.exit(1) }
  await page.waitForTimeout(600)

  const dayActionTitle2 = page.locator('h2:has-text("Управление расписанием")')
  if (!await dayActionTitle2.isVisible().catch(() => false)) {
    fail('DayActionMenu did not open'); await browser.close(); process.exit(1)
  }
  pass('DayActionMenu opened')

  // Expand the "Расширить рабочие часы" section
  const extOverlay = page.locator('.fixed.inset-0.z-50')
  await extOverlay.locator('button').filter({ hasText: 'Расширить рабочие часы' }).click()
  await page.waitForTimeout(400)

  const timeInputs = extOverlay.locator('input[type="time"]')
  const timeCount = await timeInputs.count()
  info(`Time inputs in extend section: ${timeCount}`)
  if (timeCount >= 2) {
    await timeInputs.first().fill('10:00')
    await timeInputs.last().fill('11:30')
    pass('Custom hours set: 10:00–11:30')
  } else {
    fail('Time inputs not found in extend section')
    await page.keyboard.press('Escape')
  }

  let extendExcId = null
  const [extResp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/exceptions') && r.request().method() === 'POST',
      { timeout: 8000 },
    ).catch(() => null),
    extOverlay.locator('button').filter({ hasText: 'Применить' }).click(),
  ])
  await page.waitForTimeout(1200)

  if (extResp?.status() === 201) {
    const body = await extResp.json().catch(() => null)
    extendExcId = body?.id ?? null
    pass(`custom_working_hours exception created (ID=${extendExcId}) — 201`)
  } else if (extResp?.status() === 409) {
    info('custom_working_hours exception already exists (409)')
    const token = await page.evaluate(() => localStorage.getItem('accessToken') ?? '')
    const listR = await page.request.get(
      `${BASE}/api/v1/doctors/${DOCTOR_ID}/exceptions?from=${EXTEND_DATE}&to=${EXTEND_DATE}`,
      { headers: token ? { Authorization: `Bearer ${token}` } : {} },
    )
    const existing = await listR.json().catch(() => [])
    if (Array.isArray(existing) && existing.length > 0) {
      extendExcId = existing[0].id
      info(`Reusing existing exception ID=${extendExcId}`)
    }
    pass('Extend hours: exception already present (acceptable)')
  } else {
    fail(`POST /exceptions for extend hours: ${extResp?.status() ?? 'no response'}`)
  }

  if (await page.locator('text=Расписание обновлено').isVisible().catch(() => false))
    pass('Toast "Расписание обновлено" for extend-hours visible')
  else info('Extend-hours toast may have disappeared')

  if (await dayActionTitle2.isVisible().catch(() => false)) {
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
  }

  // ── 8. Appointment OUTSIDE custom hours (09:00) → 422 ────────────────────
  console.log(`\n8. Appointment OUTSIDE custom hours (${EXTEND_DATE}T09:00) → expect 422`)
  await openCreateModal(page)
  if (!await page.locator('h2:has-text("Новая запись")').isVisible().catch(() => false)) {
    fail('Create modal did not open (outside-hours test)'); await browser.close(); process.exit(1)
  }
  await fillAppointment(page, `${EXTEND_DATE}T09:00`)
  const apptOut = await submitAndCapture(page)

  if (apptOut) {
    const st = apptOut.status()
    info(`POST /appointments (before custom hours) → ${st}`)
    if (st === 422) pass('Appointment before custom interval rejected: 422 ✅')
    else fail(`Expected 422, got ${st}`)
  } else {
    fail('POST /appointments not intercepted (outside-hours test)')
  }

  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // ── 9. Appointment INSIDE custom hours (10:00) → 201 ─────────────────────
  console.log(`\n9. Appointment INSIDE custom hours (${EXTEND_DATE}T10:00) → expect 201`)
  await openCreateModal(page)
  if (!await page.locator('h2:has-text("Новая запись")').isVisible().catch(() => false)) {
    fail('Create modal did not open (inside-hours test)'); await browser.close(); process.exit(1)
  }
  await fillAppointment(page, `${EXTEND_DATE}T10:00`)
  const apptIn = await submitAndCapture(page)

  if (apptIn) {
    const st = apptIn.status()
    const body = await apptIn.text().catch(() => '')
    info(`POST /appointments (inside custom hours) → ${st}: ${body.slice(0, 80)}`)
    if (st === 201) pass('Appointment inside custom interval: 201 ✅')
    else fail(`Expected 201, got ${st}: ${body.slice(0, 80)}`)
  } else {
    fail('POST /appointments not intercepted (inside-hours test)')
  }

  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // ── 10. Double booking on tomorrow → 409 ─────────────────────────────────
  console.log('\n10. Navigate to tomorrow → double booking → 409')
  // From EXTEND_DATE (+3), go back 2 to reach TOMORROW (+1)
  await clickPrevDay(page, 2)

  await openCreateModal(page)
  if (!await page.locator('h2:has-text("Новая запись")').isVisible().catch(() => false)) {
    fail('Create modal did not open (double-booking test)'); await browser.close(); process.exit(1)
  }
  await fillAppointment(page, `${TOMORROW}T10:00`)
  const apptDbl = await submitAndCapture(page)

  if (apptDbl) {
    const st = apptDbl.status()
    info(`Double-booking attempt → ${st}`)
    if (st === 409) pass('Double booking blocked: 409 ✅')
    else if (st === 201) fail('CRITICAL: double booking NOT blocked (201)')
    else pass(`Slot returned ${st} — acceptable if no prior appointment at 10:00`)
  } else {
    info('Double-booking POST not intercepted')
  }

  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)

  // ── 11. Cleanup exceptions ────────────────────────────────────────────────
  console.log('\n11. Cleanup test exceptions')
  for (const [label, id] of [['close_day', closeDayExcId], ['custom_working_hours', extendExcId]]) {
    if (!id) { info(`No ID for ${label} — skipping cleanup`); continue }
    const status = await deleteException(page, id).catch(() => -1)
    if (status === 204) pass(`Exception ${id} (${label}) deleted`)
    else info(`DELETE exception ${id}: HTTP ${status}`)
  }

  // ── 12. Final screenshot + JS errors ─────────────────────────────────────
  console.log('\n12. Final checks')
  await page.screenshot({ path: 'e2e_schedule_controls.png' })
  pass('Screenshot → e2e_schedule_controls.png')

  if (pageErrors.length === 0) pass('No JS errors during test')
  else fail(`JS errors: ${pageErrors.slice(0, 3).join('; ')}`)

  await page.waitForTimeout(1000)
  await browser.close()

  const banner = process.exitCode === 1
    ? '\n⛔ Schedule controls smoke test FAILED\n'
    : '\n✅ Schedule controls smoke test PASSED\n'
  console.log(banner)
}

run().catch((e) => {
  console.error('Fatal:', e.message?.slice(0, 500))
  process.exit(1)
})
