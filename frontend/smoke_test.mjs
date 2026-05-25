import { chromium } from 'playwright'

const BASE = 'http://localhost:5173'
const EXEC = 'C:/Users/User/AppData/Local/ms-playwright/chromium-1223/chrome-win64/chrome.exe'

const pass = (msg) => console.log(`  ✅ ${msg}`)
const fail = (msg) => { console.error(`  ❌ ${msg}`); process.exitCode = 1 }

async function goto(page, path) {
  await page.goto(`${BASE}${path}`)
  await page.waitForLoadState('domcontentloaded')
  await page.waitForTimeout(700)
}

async function checkPageTitle(page, expected, label) {
  try {
    const found = await page.getByRole('heading', { name: expected, exact: false }).first().isVisible({ timeout: 6000 })
    if (found) pass(label)
    else fail(`${label} — heading "${expected}" not visible`)
  } catch {
    fail(`${label} — heading "${expected}" not found`)
  }
}

async function checkVisible(page, locator, label) {
  try {
    await locator.waitFor({ timeout: 6000 })
    pass(label)
  } catch {
    fail(`${label} — not visible`)
  }
}

async function checkInvisible(page, text, label) {
  const count = await page.getByText(text, { exact: false }).count()
  if (count === 0) pass(label)
  else fail(`${label} — text "${text}" still present (count=${count})`)
}

async function run() {
  const browser = await chromium.launch({ headless: true, executablePath: EXEC })
  const context = await browser.newContext()

  // Seed auth BEFORE any navigation — addInitScript runs before page scripts
  await context.addInitScript(() => {
    localStorage.setItem('accessToken', 'smoke_token')
    localStorage.setItem('refreshToken', 'smoke_refresh')
    localStorage.setItem('user', JSON.stringify({ id: 1, role: 'admin', email: 'smoke@test.com' }))
  })

  const page = await context.newPage()

  // Mock all API calls so the fake token never hits the backend.
  // Without this, every request returns 401 → refresh fails → clearTokens() → redirect to /login.
  await page.route('**/api/v1/**', async (route) => {
    const pathname = new URL(route.request().url()).pathname
    if (pathname.includes('/auth/refresh')) {
      return route.fulfill({ status: 200, contentType: 'application/json',
        body: JSON.stringify({ access_token: 'smoke_token', refresh_token: 'smoke_refresh' }) })
    }
    if (pathname.includes('/auth/me')) {
      return route.fulfill({ status: 200, contentType: 'application/json',
        body: JSON.stringify({ id: 1, role: 'admin', email: 'smoke@test.com' }) })
    }
    // All list/detail endpoints: return empty array (pages must tolerate empty state)
    return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
  })

  // ─── 1. /admin/services ──────────────────────────────────────────────────
  console.log('\n1. /admin/services')
  await goto(page, '/admin/services')
  await checkPageTitle(page, 'Услуги', 'page heading "Услуги"')
  const addBtn = page.getByRole('button', { name: 'Добавить услугу' })
  await checkVisible(page, addBtn, '"Добавить услугу" button')
  await addBtn.click()
  await page.waitForTimeout(400)
  await checkVisible(page, page.getByText('Новая услуга'), 'modal title "Новая услуга"')
  const dirField = await page.getByText('Направление').count()
  if (dirField === 0) pass('No "Направление" field in form (removed)')
  else fail('"Направление" field still present in form')
  await checkVisible(page, page.getByText('Категория'), 'Category field present')
  await checkVisible(page, page.getByText('Код услуги'), 'Code field present')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)

  // ─── 2. /admin/patients ──────────────────────────────────────────────────
  console.log('\n2. /admin/patients')
  await goto(page, '/admin/patients')
  await checkPageTitle(page, 'Пациенты', 'page heading "Пациенты"')
  const addPatBtn = page.getByRole('button', { name: 'Добавить пациента' })
  await checkVisible(page, addPatBtn, '"Добавить пациента" button')
  await checkVisible(page, page.locator('select'), 'Source filter dropdown')
  await addPatBtn.click()
  await page.waitForTimeout(400)
  await checkVisible(page, page.getByText('Новый пациент'), 'Create patient modal title')
  await checkVisible(page, page.locator('input[placeholder="Иванов Иван Иванович"]'), 'Full name input')
  await checkVisible(page, page.locator('input[type="date"]'), 'DOB date input')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)

  // ─── 3. /admin/schedule-grid ─────────────────────────────────────────────
  console.log('\n3. /admin/schedule-grid')
  await goto(page, '/admin/schedule-grid')
  await checkVisible(page, page.getByRole('button', { name: 'Сегодня' }), '"Сегодня" button in left panel')
  await checkVisible(page, page.locator('input[placeholder="ФИО или телефон…"]'), 'Patient search input')
  await checkVisible(page, page.locator('.flex.h-full').first(), 'Grid page layout loaded')
  await checkVisible(page, page.getByRole('button', { name: 'Новая запись' }), '"Новая запись" button')

  // ─── 4. /admin/settings/clinic ───────────────────────────────────────────
  console.log('\n4. /admin/settings/clinic')
  await goto(page, '/admin/settings/clinic')
  await checkPageTitle(page, 'Профиль клиники', 'page heading "Профиль клиники"')
  const editBtn = page.getByRole('button', { name: 'Редактировать' })
  await checkVisible(page, editBtn, '"Редактировать" button')
  await checkInvisible(page, 'обратитесь к администратору', 'No "contact admin" text')
  await editBtn.click()
  await page.waitForTimeout(300)
  await checkVisible(page, page.getByRole('button', { name: 'Сохранить' }), '"Сохранить" button in edit mode')
  await checkVisible(page, page.locator('input[placeholder="МЕДИК-ПРОФИ"]'), 'Clinic name input editable')
  await page.getByRole('button', { name: 'Сохранить' }).click()
  await page.waitForTimeout(300)
  await checkVisible(page, page.getByRole('button', { name: 'Редактировать' }), 'Returns to view mode after save')

  // ─── 5. /admin/referrers ─────────────────────────────────────────────────
  console.log('\n5. /admin/referrers')
  await page.evaluate(() => localStorage.removeItem('referrers_v1'))
  await goto(page, '/admin/referrers')
  await checkPageTitle(page, 'Внешние направители', 'page heading "Внешние направители"')
  const addRefBtn = page.getByRole('button', { name: 'Добавить направителя' })
  await checkVisible(page, addRefBtn, '"Добавить направителя" button')
  await addRefBtn.click()
  await page.waitForTimeout(400)
  await checkVisible(page, page.getByText('Новый направитель'), 'Create referrer modal title')
  await checkVisible(page, page.locator('input[placeholder="Иванов А.П."]'), 'Referrer name input')
  await checkVisible(page, page.getByText('Комиссия за услуги (%)'), 'Commission service field')
  await checkVisible(page, page.getByText('Комиссия за лабораторию (%)'), 'Commission lab field')
  await page.fill('input[placeholder="Иванов А.П."]', 'Тестовый Врач')
  await page.getByRole('button', { name: 'Сохранить' }).click()
  await page.waitForTimeout(500)
  await checkVisible(page, page.getByText('Тестовый Врач'), 'Referrer appears in table after create')
  const rowBtns = page.locator('table tbody tr button')
  await rowBtns.nth(1).click()
  await page.waitForTimeout(300)
  await checkVisible(page, page.getByText('Удалить направителя'), 'Delete confirm dialog appears')
  await page.keyboard.press('Escape')

  // ─── 6. /admin/reports ───────────────────────────────────────────────────
  console.log('\n6. /admin/reports')
  await goto(page, '/admin/reports')
  await checkPageTitle(page, 'Отчёты', 'page heading "Отчёты"')
  const chips = [
    'По врачам', 'По услугам', 'По лаборатории',
    'По внешним направителям', 'Кому сколько выплатить',
    'По администраторам', 'Касса', 'Средний чек', 'Записи / отмены / неявки',
  ]
  for (const chip of chips) {
    const visible = await page.getByRole('button', { name: chip }).isVisible().catch(() => false)
    if (visible) pass(`chip "${chip}"`)
    else fail(`chip "${chip}" not visible`)
  }

  await browser.close()

  console.log(process.exitCode === 1
    ? '\n⛔ Smoke tests FAILED — see failures above\n'
    : '\n✅ All smoke tests passed\n')
}

run().catch((e) => {
  console.error('Fatal:', e.message?.slice(0, 400))
  process.exit(1)
})
