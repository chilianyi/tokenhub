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
import { useQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import {
  DataTablePage,
  useDataTable,
} from '@/components/data-table'
import { useMediaQuery } from '@/hooks'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import { createServerError } from '@/lib/server-error-message'

import { getOrganizations } from '../api'
import type { Organization } from '../types'
import { useOrganizationsColumns } from './organizations-columns'
import { useOrganizations } from './organizations-provider'

const route = getRouteApi('/_authenticated/organizations/')

export function OrganizationsTable() {
  const { t } = useTranslation()
  const columns = useOrganizationsColumns()
  const { refreshTrigger } = useOrganizations()
  const isMobile = useMediaQuery('(max-width: 640px)')

  const { globalFilter, pagination, ensurePageInRange } = useTableUrlState({
    search: route.useSearch(),
    navigate: route.useNavigate(),
    pagination: { defaultPage: 1, defaultPageSize: isMobile ? 10 : 20 },
    globalFilter: { enabled: true, key: 'filter' },
  })

  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      'organizations',
      pagination.pageIndex + 1,
      pagination.pageSize,
      globalFilter,
      refreshTrigger,
    ],
    queryFn: async () => {
      const result = await getOrganizations({
        keyword: globalFilter,
        p: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
      })
      if (!result.success) {
        throw createServerError(result, t('Failed to load organizations'))
      }
      return result.data
    },
    placeholderData: (prev) => prev,
  })

  const { table } = useDataTable<Organization>({
    data: data?.items ?? [],
    columns,
    getRowId: (row) => String(row.id),
    globalFilter,
    pagination,
    manualPagination: true,
    manualFiltering: true,
    totalCount: data?.total ?? 0,
    ensurePageInRange,
  })

  return (
    <DataTablePage
      table={table}
      columns={columns}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle={t('No organizations yet')}
      emptyDescription={t(
        'Create an organization to cap the total quota you can allocate to its members.'
      )}
      skeletonKeyPrefix='organizations-skeleton'
      applyHeaderSize
      toolbarProps={{
        searchPlaceholder: t('Search organizations...'),
        searchDebounceMs: 500,
      }}
    />
  )
}
