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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { handleServerError } from '@/lib/handle-server-error'

import { deleteOrganization } from '../api'
import { useOrganizations } from './organizations-provider'

export function OrganizationsDeleteDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const { currentRow, triggerRefresh } = useOrganizations()
  const [isDeleting, setIsDeleting] = useState(false)

  const onConfirm = async () => {
    if (!currentRow) return
    setIsDeleting(true)
    try {
      const result = await deleteOrganization(currentRow.id)
      if (result.success) {
        toast.success(t('Organization deleted'))
        onOpenChange(false)
        triggerRefresh()
      } else {
        handleServerError(result)
      }
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {t('Delete organization "{{name}}"?', {
              name: currentRow?.name ?? '',
            })}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              'Members will be detached from the organization. Their personal balances are not affected and they can keep using the gateway.'
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isDeleting}>
            {t('Cancel')}
          </AlertDialogCancel>
          <AlertDialogAction
            onClick={(e) => {
              e.preventDefault()
              void onConfirm()
            }}
            disabled={isDeleting}
          >
            {isDeleting ? t('Deleting...') : t('Delete')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
