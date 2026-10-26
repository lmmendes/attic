import { expect, test } from '@playwright/test'

const apiURL = (path: string) => `${process.env.E2E_API_BASE || process.env.E2E_BASE_URL || 'http://127.0.0.1:3000'}${path}`

for (const viewport of [
  { width: 393, height: 600 },
  { width: 740, height: 360 },
  { width: 393, height: 340 }
]) {
  test(`location dialog stays usable at ${viewport.width}×${viewport.height}`, async ({ page }, testInfo) => {
    await page.goto('/login')
    await page.getByPlaceholder('Enter your email').fill(process.env.E2E_EMAIL || 'admin')
    await page.getByPlaceholder('Enter your password').fill(process.env.E2E_PASSWORD || 'admin')
    await page.getByRole('button', { name: 'Sign in', exact: true }).click()
    await expect(page).toHaveURL(url => url.pathname === '/')
    await page.setViewportSize(viewport)
    await page.goto('/locations')
    const name = `Mobile location ${Date.now()} ${viewport.height}`
    let id = ''
    try {
      await page.getByRole('button', { name: 'Add Location', exact: true }).first().click()
      const create = page.getByRole('dialog', { name: 'New Location', exact: true })
      const form = create.locator('form')
      const createButton = create.getByRole('button', { name: 'Create', exact: true })
      const cancel = create.getByRole('button', { name: 'Cancel', exact: true })
      await expect(createButton).toBeInViewport({ ratio: 1 })
      await expect(cancel).toBeInViewport({ ratio: 1 })
      await expect.poll(() => form.evaluate(element => element.scrollHeight > element.clientHeight)).toBe(true)
      await form.evaluate(element => element.scrollTo(0, element.scrollHeight))
      await expect(create.getByRole('combobox')).toBeInViewport({ ratio: 1 })
      await expect(createButton).toBeInViewport({ ratio: 1 })
      await expect(cancel).toBeInViewport({ ratio: 1 })
      await cancel.click()
      await expect(create).not.toBeVisible()
      await page.getByRole('button', { name: 'Add Location', exact: true }).first().click()
      await create.getByPlaceholder('Location name').fill(name)
      await createButton.click()
      await expect(create).not.toBeVisible()
      const locations = await (await page.request.get(apiURL('/api/locations'))).json() as Array<{ id: string, name: string }>
      id = locations.find(item => item.name === name)!.id
      await page.getByRole('button', { name: 'Edit location', exact: true }).click()
      const edit = page.getByRole('dialog', { name: 'Edit Location', exact: true })
      const update = edit.getByRole('button', { name: 'Update', exact: true })
      await expect(update).toBeInViewport({ ratio: 1 })
      await expect(edit.getByRole('button', { name: 'Cancel', exact: true })).toBeInViewport({ ratio: 1 })
      await edit.getByPlaceholder('Location name').fill(`${name} updated`)
      await update.click()
      await expect(edit).not.toBeVisible()
      await expect(page.getByRole('button', { name: `View ${name} updated`, exact: true })).toBeVisible()
    } finally {
      testInfo.setTimeout(testInfo.timeout + 10_000)
      if (id) await page.request.delete(apiURL(`/api/locations/${id}`))
    }
  })
}
