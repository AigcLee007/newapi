/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, test, vi } from 'vitest'

import { getImageAsyncTaskDetail } from '../../api'
import type { ImageAsyncTaskDetail } from '../../types'
import { ImageAsyncTaskDialog } from '../dialogs/image-async-task-dialog'

vi.mock('../../api', () => ({ getImageAsyncTaskDetail: vi.fn() }))

const detail: ImageAsyncTaskDetail = {
  task_id: 'image-task-1',
  user_id: 1,
  channel_id: 64,
  status: 'SUCCESS',
  submit_time: 1,
  start_time: 1,
  finish_time: 2,
  quota: 0,
  result_objects: [{ index: 0, url: 'https://images.example/stored.png' }],
  data: {
    data: [
      { url: 'https://images.example/stored.png' },
      { url: 'https://images.example/upstream.png' },
    ],
  },
}

function renderDialog(open = true) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ImageAsyncTaskDialog
        taskId={detail.task_id}
        open={open}
        onOpenChange={vi.fn()}
      />
    </QueryClientProvider>
  )
}

test('successful task displays unique stored and upstream image links', async () => {
  vi.mocked(getImageAsyncTaskDetail).mockResolvedValue({
    success: true,
    data: detail,
  })
  renderDialog()
  const links = await screen.findAllByRole('link', { name: 'Open image' })
  expect(links.map((link) => link.getAttribute('href'))).toEqual([
    'https://images.example/stored.png',
    'https://images.example/upstream.png',
  ])
  expect(
    screen.getAllByRole('button', { name: 'Copy URL' })[0]
  ).toHaveAttribute('aria-label', 'Copy URL')
})

test('copying a returned image link writes the exact URL to the clipboard', async () => {
  const user = userEvent.setup()
  vi.mocked(getImageAsyncTaskDetail).mockResolvedValue({
    success: true,
    data: detail,
  })
  renderDialog()
  const buttons = await screen.findAllByRole('button', { name: 'Copy URL' })
  await user.click(buttons[0] as HTMLElement)
  expect(await navigator.clipboard.readText()).toBe(
    'https://images.example/stored.png'
  )
})

test('failed detail request shows the server reason', async () => {
  vi.mocked(getImageAsyncTaskDetail).mockResolvedValue({
    success: false,
    message: 'Task is unavailable',
  })
  renderDialog()
  expect(await screen.findByText('Task is unavailable')).toBeVisible()
})

test('successful task without image URLs shows the empty result message', async () => {
  vi.mocked(getImageAsyncTaskDetail).mockResolvedValue({
    success: true,
    data: { ...detail, result_objects: [], data: {} },
  })
  renderDialog()
  expect(
    await screen.findByText('No image URL is available yet.')
  ).toBeVisible()
})
