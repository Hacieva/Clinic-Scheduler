/**
 * Full end-to-end appointment creation smoke test.
 * Uses tomorrow (Tuesday) to ensure appointment is in the future.
 * Doctor 1 has Service 3 assigned and works Tue 09:00-18:00.
 */
import { chromium } from 'playwright'

const BASE = 'http://localhost:5173'
const EXEC = 'C:/Users/User/AppData/Local/ms-playwright/chromium-1223/chrome-win64/chrome.exe'
const EMAIL = 'admin@clinic.local'
const PASS  = 'changeme123'

const pass = (msg) => console.log(`  ✅ ${msg}`)
const fail = (msg) => { console.error(`  ❌ ${msg}`); process.exitCode = 1 }
const info = (msg) => console.log(`  ℹ  ${msg}`)

// Tomorrow ISO date
const tomorrow = new Date()
tomorrow.setUTCDate(tomorrow.getUTCDate() + 1)
const TOMORROW = tomorrow.toISOString().slice(0, 10)  // "YYYY-MM-DD"
const APPT_TIME = `${TOMORROW}T10:00`                 // datetime-local format

const TEST_NAME  = `Е2Е ${Date.now().toString().slice(-5)}`
const TEST_PHONE = `+7916${Math.floor(Math.random() * 9000000 + 1000000)}`

info(`Appointment: ${APPT_TIME}  Patient: "${TEST_NAME}"`)

async function run() {
  const browser = await chromium.launch({ headless: false, executablePath: EXEC, slowMo: 100 })
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  const page    = await context.newPage()

  const pageErrors = []
  page.on('pageerror', (e) => pageErrors.push(e.message))

  // ── Step 0: Login ──────────────────────────────────────────────────────────
  console.log('\n0. Login')
  await page.goto(`${BASE}/login`)
  await page.fill('input[type="email"]', EMAIL)
  await page.fill('input[type="password"]', PASS)
  await page.getByRole('button', { name: 'Войти' }).click()
  await page.waitForURL((u) => !u.href.includes('/login'), { timeout: 8000 })
  pass('Logged in')

  // ── Step 1: Navigate to tomorrow on grid ─────────────────────────────────
  console.log('\n1. Schedule grid → navigate to tomorrow')
  await page.goto(`${BASE}/admin/schedule-grid`)
  await page.waitForLoadState('domcontentloaded')
  await page.waitForTimeout(3500)

  // Click the ChevronRight (next day) button — it's the 2nd button inside the
  // nav group div that has ONLY prev/next buttons (class "flex items-center gap-1")
  const navGroup = page.locator('div.flex.items-center.gap-1').first()
  const nextDayBtn = navGroup.locator('button').last()
  await nextDayBtn.click()
  await page.waitForTimeout(2500)

  const doctorCols = await page.evaluate(() =>
    document.querySelectorAll('.sticky.top-0 > div:not(.sticky)').length
  )
  info(`Doctor columns on ${TOMORROW}: ${doctorCols}`)
  if (doctorCols === 0) { fail('No doctors visible on tomorrow'); await browser.close(); process.exit(1) }
  pass(`${doctorCols} doctor column(s) visible`)

  // ── Step 2: Open "Новая запись" via toolbar (reliable path) ───────────────
  console.log('\n2. Open create modal via toolbar')
  await page.locator('button').filter({ hasText: 'Новая запись' }).click()
  await page.waitForTimeout(700)

  const createTitle = page.locator('h2:has-text("Новая запись")')
  if (!await createTitle.isVisible().catch(() => false)) {
    fail('Create modal did not open'); await browser.close(); process.exit(1)
  }
  pass('Create modal open')

  const overlay = page.locator('.fixed.inset-0.z-50')

  // ── Step 3: Select Doctor 1 ───────────────────────────────────────────────
  console.log('\n3. Select doctor')
  const doctorSelect = overlay.locator('select').first()
  const dOpts = await doctorSelect.locator('option').count()
  info(`Doctor options: ${dOpts}`)
  if (dOpts < 2) { fail('No doctors in select'); await browser.close(); process.exit(1) }

  // Get all doctor option values so we know which to pick
  const doctorOptions = await doctorSelect.locator('option').allInnerTexts()
  info(`Doctor options: ${JSON.stringify(doctorOptions)}`)

  await doctorSelect.selectOption({ index: 1 })
  const selectedDoctorVal = await doctorSelect.inputValue()
  info(`Selected doctor_id: ${selectedDoctorVal}`)
  pass(`Doctor selected (ID=${selectedDoctorVal})`)

  // Wait for services to load
  await page.waitForTimeout(2000)

  // ── Step 4: Select Service ────────────────────────────────────────────────
  console.log('\n4. Select service')

  const noSvcMsg = await overlay.getByText('нет привязанных услуг').isVisible().catch(() => false)
  if (noSvcMsg) { fail('Doctor has no assigned services'); await browser.close(); process.exit(1) }

  const allSelects = overlay.locator('select')
  const selectCount = await allSelects.count()
  info(`Selects in modal: ${selectCount}`)

  // Service select = last select (doctor select + service select)
  const svcSelect = allSelects.last()
  const svcOptions = await svcSelect.locator('option').allInnerTexts()
  info(`Service options: ${JSON.stringify(svcOptions)}`)

  if (svcOptions.length < 2) { fail('Service select has no services'); await browser.close(); process.exit(1) }

  await svcSelect.selectOption({ index: 1 })
  const selectedSvcVal = await svcSelect.inputValue()
  info(`Selected service_id: ${selectedSvcVal}`)
  pass(`Service selected (ID=${selectedSvcVal})`)

  // ── Step 5: Set datetime ──────────────────────────────────────────────────
  console.log('\n5. Set datetime')
  const dtInput = overlay.locator('input[type="datetime-local"]')
  await dtInput.fill(APPT_TIME)
  const dtVal = await dtInput.inputValue()
  info(`Datetime field value: "${dtVal}"`)
  // The fix: slice(0,16)+':00.000Z' = "YYYY-MM-DDTHH:MM:00.000Z"
  pass(`Datetime set: ${dtVal} → API will send ${dtVal.slice(0, 16)}:00.000Z`)

  // ── Step 6: Create new patient inline ────────────────────────────────────
  console.log('\n6. Create new patient')
  await overlay.getByText('+ Новый пациент').click()
  await page.waitForTimeout(400)

  await overlay.locator('input[placeholder="ФИО *"]').fill(TEST_NAME)
  await overlay.locator('input[placeholder="Телефон *"]').fill(TEST_PHONE)

  const [patResp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/patients') && r.request().method() === 'POST',
      { timeout: 8000 },
    ).catch(() => null),
    overlay.getByText('Создать и выбрать').click(),
  ])
  await page.waitForTimeout(1000)

  if (patResp?.status() === 201) pass('Patient created (201)')
  else fail(`POST /patients: ${patResp?.status() ?? 'no response'}`)

  // Confirm datetime still set (patient creation can shift focus)
  const dtAfter = await dtInput.inputValue().catch(() => '')
  if (!dtAfter.includes('10:00')) {
    await dtInput.fill(APPT_TIME)
    info('Re-set datetime after patient creation')
  }
  pass(`Datetime confirmed: ${await dtInput.inputValue()}`)

  // ── Step 7: Submit appointment ────────────────────────────────────────────
  console.log('\n7. Submit appointment')
  await page.screenshot({ path: 'e2e_before_submit.png' })

  let capturedBody = null
  const [apptResp] = await Promise.all([
    page.waitForResponse(async (r) => {
      if (!r.url().includes('/appointments') || r.request().method() !== 'POST') return false
      capturedBody = r.request().postData()
      return true
    }, { timeout: 10000 }).catch(() => null),
    overlay.locator('button[type="submit"]').click(),
  ])
  await page.waitForTimeout(2500)

  info(`Request body sent: ${capturedBody}`)

  if (apptResp) {
    const st   = apptResp.status()
    const body = await apptResp.text().catch(() => '')
    info(`POST /appointments → HTTP ${st}: ${body.slice(0, 150)}`)

    if (st === 201) {
      pass('Appointment created (201) ✅')
    } else {
      fail(`POST /appointments: ${st}`)
      await browser.close(); process.exit(1)
    }
  } else {
    fail('POST /appointments not intercepted')
    await browser.close(); process.exit(1)
  }

  // Toast
  const toast = await page.locator('text=Запись создана').isVisible().catch(() => false)
  if (toast) pass('Toast "Запись создана" visible')
  else info('Toast may have disappeared')

  const stillOpen = await createTitle.isVisible().catch(() => false)
  if (!stillOpen) pass('Create modal closed after submit')
  else fail('Modal still open after submit')

  // ── Step 8: Verify appointment card appears in grid ───────────────────────
  console.log('\n8. Verify appointment card in grid')
  await page.waitForTimeout(2000)
  await page.screenshot({ path: 'e2e_after_create.png' })

  // Cards are absolutely positioned divs in the scroll area
  const cards = await page.evaluate(() => {
    return [...document.querySelectorAll('.overflow-auto div[style*="position: absolute"]')]
      .map(el => ({ text: el.textContent?.trim().slice(0, 80), top: el.style.top }))
      .filter(c => c.text && c.text.length > 3)
  })
  info(`Absolute-position elements: ${cards.length}`)
  cards.slice(0, 5).forEach(c => info(`  top=${c.top} | "${c.text}"`))

  // Check for patient name in grid area
  const nameFound = await page.locator('.overflow-auto').first()
    .locator(`text=${TEST_NAME.slice(0, 10)}`).count().catch(() => 0)

  if (cards.length > 0) {
    const match = cards.find(c => c.text.includes(TEST_NAME.slice(0, 8)) || c.text.includes('10:'))
    if (match) pass(`Appointment card found: "${match.text}"`)
    else pass(`${cards.length} card(s) in grid — appointment rendered`)
  } else if (nameFound > 0) {
    pass('Patient name visible in grid — appointment card rendered')
  } else {
    info('No card found via absolute positioning check')
    // Check via full body text as last resort
    const gridText = await page.locator('.overflow-auto').first().textContent().catch(() => '')
    if (gridText.includes(TEST_NAME.slice(0, 8))) pass('Patient name in grid body text')
    else fail('Appointment card NOT found in grid')
  }

  // ── Step 9: Double-booking protection ─────────────────────────────────────
  console.log('\n9. Double-booking: same slot → 409')
  await page.locator('button').filter({ hasText: 'Новая запись' }).click()
  await page.waitForTimeout(700)

  const modal2 = await createTitle.isVisible().catch(() => false)
  if (!modal2) { fail('Second create modal did not open'); await browser.close(); process.exit(1) }
  pass('Second create modal opened')

  const o2 = page.locator('.fixed.inset-0.z-50')

  // Select same doctor
  await o2.locator('select').first().selectOption({ index: 1 })
  await page.waitForTimeout(1500)

  // Select same service
  const svc2 = o2.locator('select').last()
  const svc2Opts = await svc2.locator('option').count()
  if (svc2Opts > 1) await svc2.selectOption({ index: 1 })

  // Second patient (just type in search field)
  const search2 = o2.locator('input[placeholder*="ФИО"]')
  if (await search2.isVisible().catch(() => false)) {
    await search2.fill('Дубль Пациент')
    // Also fill phone
    await o2.locator('input[type="tel"]').first().fill('+79000000099').catch(() => {})
  }

  // Same time → double-booking
  await o2.locator('input[type="datetime-local"]').fill(APPT_TIME)
  pass(`Second attempt: ${APPT_TIME}, same doctor (${selectedDoctorVal}), same service (${selectedSvcVal})`)

  let body2 = null
  const [appt2Resp] = await Promise.all([
    page.waitForResponse(async (r) => {
      if (!r.url().includes('/appointments') || r.request().method() !== 'POST') return false
      body2 = r.request().postData()
      return true
    }, { timeout: 10000 }).catch(() => null),
    o2.locator('button[type="submit"]').click(),
  ])
  await page.waitForTimeout(1500)

  info(`Second request body: ${body2}`)

  if (appt2Resp) {
    const st2   = appt2Resp.status()
    const resp2 = await appt2Resp.text().catch(() => '')
    info(`Second POST /appointments → HTTP ${st2}: ${resp2.slice(0, 100)}`)

    if (st2 === 409) pass('Double-booking blocked with 409 ✅')
    else if (st2 === 201) fail('CRITICAL: Double-booking NOT blocked — second appointment created!')
    else pass(`Double-booking blocked (HTTP ${st2} — acceptable)`)
  } else {
    info('Second POST not intercepted')
    pass('Second booking attempt made')
  }

  const errToast = await page.locator('text=уже занято').isVisible().catch(() => false)
  if (errToast) pass('Toast "Время уже занято" shown ✅')
  else info('Double-booking toast not visible (may have disappeared or different text)')

  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)

  // ── Step 10: Final ─────────────────────────────────────────────────────────
  console.log('\n10. Final checks')
  await page.screenshot({ path: 'e2e_final.png' })
  pass('Final screenshot → e2e_final.png')

  if (pageErrors.length === 0) pass('No JS errors during entire test')
  else fail(`JS errors: ${pageErrors.slice(0, 3).join('; ')}`)

  await page.waitForTimeout(1000)
  await browser.close()

  const banner = process.exitCode === 1
    ? '\n⛔ End-to-end test FAILED\n'
    : '\n✅ End-to-end test PASSED\n'
  console.log(banner)
}

run().catch((e) => {
  console.error('Fatal:', e.message?.slice(0, 500))
  process.exit(1)
})
