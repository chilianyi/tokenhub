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
import React, { useState } from 'react'

import useDialogState from '@/hooks/use-dialog'

import type { Organization } from '../types'

export type OrganizationsDialogType = 'create' | 'update' | 'delete'

type OrganizationsContextType = {
  open: OrganizationsDialogType | null
  setOpen: (str: OrganizationsDialogType | null) => void
  currentRow: Organization | null
  setCurrentRow: React.Dispatch<React.SetStateAction<Organization | null>>
  refreshTrigger: number
  triggerRefresh: () => void
}

const OrganizationsContext =
  React.createContext<OrganizationsContextType | null>(null)

export function OrganizationsProvider({
  children,
}: {
  children: React.ReactNode
}) {
  const [open, setOpen] = useDialogState<OrganizationsDialogType>(null)
  const [currentRow, setCurrentRow] = useState<Organization | null>(null)
  const [refreshTrigger, setRefreshTrigger] = useState(0)

  const triggerRefresh = () => setRefreshTrigger((prev) => prev + 1)

  return (
    <OrganizationsContext
      value={{
        open,
        setOpen,
        currentRow,
        setCurrentRow,
        refreshTrigger,
        triggerRefresh,
      }}
    >
      {children}
    </OrganizationsContext>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useOrganizations() {
  const context = React.useContext(OrganizationsContext)
  if (!context) {
    throw new Error(
      'useOrganizations must be used within OrganizationsProvider'
    )
  }
  return context
}
