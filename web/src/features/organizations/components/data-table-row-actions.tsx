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
import type { Row } from '@tanstack/react-table'
import { Edit, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { DataTableRowActionMenu } from '@/components/data-table/core/row-action-menu'
import { DropdownMenuItem } from '@/components/ui/dropdown-menu'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import { organizationSchema } from '../types'
import { useOrganizations } from './organizations-provider'

interface DataTableRowActionsProps<TData> {
  row: Row<TData>
}

export function DataTableRowActions<TData>({
  row,
}: DataTableRowActionsProps<TData>) {
  const { t } = useTranslation()
  const organization = organizationSchema.parse(row.original)
  const { setOpen, setCurrentRow } = useOrganizations()

  return (
    <DataTableRowActionMenu ariaLabel={t('Open menu')}>
      <Tooltip>
        <TooltipTrigger render={<span />}>
          <DropdownMenuItem
            onClick={() => {
              setCurrentRow(organization)
              setOpen('update')
            }}
          >
            <Edit />
            {t('Edit')}
          </DropdownMenuItem>
        </TooltipTrigger>
        <TooltipContent>{t('Edit organization')}</TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger render={<span />}>
          <DropdownMenuItem
            variant='destructive'
            onClick={() => {
              setCurrentRow(organization)
              setOpen('delete')
            }}
          >
            <Trash2 />
            {t('Delete')}
          </DropdownMenuItem>
        </TooltipTrigger>
        <TooltipContent>
          {t(
            'Deleting an organization detaches all its members; their balances are unaffected.'
          )}
        </TooltipContent>
      </Tooltip>
    </DataTableRowActionMenu>
  )
}
