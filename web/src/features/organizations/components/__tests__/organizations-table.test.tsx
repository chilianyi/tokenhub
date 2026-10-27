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
import { render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, test, vi } from 'vitest'

import { OrganizationsTable } from '../organizations-table'
import { OrganizationsProvider } from '../organizations-provider'

// 回归测试：OrganizationsTable 曾因 useDataTable 缺少 onPaginationChange /
// onGlobalFilterChange 而在渲染时崩溃（table.getPageCount() 读到 undefined pageSize）。
// 这里只验证「能渲染出空态且不抛错」，确保分页受控状态被正确接上。

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    getRouteApi: () => ({
      useSearch: () => ({ page: 1, pageSize: 10, filter: '' }),
      useNavigate: () => vi.fn(),
    }),
  }
})

vi.mock('@/hooks/use-table-url-state', () => ({
  useTableUrlState: ({ pagination }: { pagination: { defaultPage: number; defaultPageSize: number } }) => ({
    globalFilter: '',
    onGlobalFilterChange: vi.fn(),
    columnFilters: [],
    onColumnFiltersChange: vi.fn(),
    // 关键：pagination 必须是完整对象（pageIndex / pageSize 都存在），
    // 否则 useDataTable 计算 pageCount 时会崩。
    pagination: {
      pageIndex: pagination.defaultPage - 1,
      pageSize: pagination.defaultPageSize,
    },
    onPaginationChange: vi.fn(),
    ensurePageInRange: vi.fn(),
  }),
}))

vi.mock('@/hooks', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/hooks')>()
  return { ...actual, useMediaQuery: () => false }
})

vi.mock('../api', () => ({
  getOrganizations: vi.fn(async () => ({
    success: true,
    message: '',
    data: { page: 1, page_size: 10, total: 0, items: [] },
  })),
}))

function Wrapper({ children }: { children: ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return (
    <QueryClientProvider client={queryClient}>
      <OrganizationsProvider>{children}</OrganizationsProvider>
    </QueryClientProvider>
  )
}

describe('OrganizationsTable', () => {
  test('渲染空列表时不抛错，并展示空态文案', async () => {
    render(<OrganizationsTable />, { wrapper: Wrapper })

    await waitFor(() => {
      expect(screen.getByText(/No organizations yet/i)).toBeInTheDocument()
    })
  })
})
