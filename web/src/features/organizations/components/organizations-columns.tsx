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
import type { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { formatQuota, formatTimestampToDate } from '@/lib/format'

import type { Organization } from '../types'
import { DataTableRowActions } from './data-table-row-actions'

export function useOrganizationsColumns(): ColumnDef<Organization>[] {
  const { t } = useTranslation()
  return [
    {
      accessorKey: 'id',
      header: t('ID'),
      meta: { mobileHidden: true },
      cell: ({ row }) => (
        <TableId value={row.getValue('id') as number} className='w-[60px]' />
      ),
      enableSorting: false,
      size: 70,
    },
    {
      accessorKey: 'name',
      header: t('Name'),
      cell: ({ row }) => (
        <span className='font-medium'>{row.getValue('name') as string}</span>
      ),
      enableSorting: false,
      size: 200,
    },
    {
      accessorKey: 'quota',
      header: t('Quota Pool'),
      cell: ({ row }) => (
        <span className='font-mono text-sm'>
          {formatQuota(row.getValue('quota') as number)}
        </span>
      ),
      enableSorting: false,
      size: 140,
    },
    {
      accessorKey: 'allocated',
      header: t('Allocated'),
      cell: ({ row }) => {
        const allocated = row.getValue('allocated') as number
        const quota = row.original.quota
        const pct = quota > 0 ? (allocated / quota) * 100 : 0
        return (
          <div className='flex flex-col gap-0.5'>
            <span className='font-mono text-sm'>{formatQuota(allocated)}</span>
            <span className='text-muted-foreground text-xs'>
              {pct.toFixed(0)}%
            </span>
          </div>
        )
      },
      enableSorting: false,
      size: 140,
    },
    {
      id: 'remaining',
      header: t('Remaining'),
      cell: ({ row }) => {
        const remaining = row.original.quota - row.original.allocated
        return (
          <span
            className={
              remaining > 0
                ? 'font-mono text-sm'
                : 'text-destructive font-mono text-sm'
            }
          >
            {formatQuota(remaining)}
          </span>
        )
      },
      enableSorting: false,
      size: 140,
    },
    {
      accessorKey: 'status',
      header: t('Status'),
      cell: ({ row }) => {
        const status = row.getValue('status') as number
        return (
          <StatusBadge
            label={status === 1 ? t('Enabled') : t('Disabled')}
            variant={status === 1 ? 'success' : 'neutral'}
            size='sm'
            showDot
          />
        )
      },
      enableSorting: false,
      size: 100,
    },
    {
      accessorKey: 'remark',
      header: t('Remark'),
      meta: { mobileHidden: true },
      cell: ({ row }) => (
        <span className='text-muted-foreground text-sm'>
          {(row.getValue('remark') as string) || '-'}
        </span>
      ),
      enableSorting: false,
      size: 200,
    },
    {
      accessorKey: 'created_time',
      header: t('Created At'),
      meta: { mobileHidden: true },
      cell: ({ row }) => (
        <span className='text-muted-foreground text-sm'>
          {formatTimestampToDate(row.getValue('created_time') as number)}
        </span>
      ),
      enableSorting: false,
      size: 160,
    },
    {
      id: 'actions',
      cell: ({ row }) => <DataTableRowActions row={row} />,
      enableSorting: false,
      enableHiding: false,
      size: 60,
    },
  ]
}
