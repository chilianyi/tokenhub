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
import { api } from '@/lib/api'

import type {
  ApiResponse,
  Organization,
  OrganizationFormValues,
  PaginatedOrganizations,
  SearchOrganizationsParams,
} from './types'

// ============================================================================
// Organization API
// ============================================================================

export async function getOrganizations(
  params: SearchOrganizationsParams = {}
): Promise<ApiResponse<PaginatedOrganizations>> {
  const { keyword = '', p = 1, page_size = 10 } = params
  const query = new URLSearchParams()
  if (keyword) query.set('keyword', keyword)
  query.set('p', String(p))
  query.set('page_size', String(page_size))
  const res = await api.get(`/api/organizations/?${query.toString()}`)
  return res.data
}

export async function getOrganization(
  id: number
): Promise<ApiResponse<Organization>> {
  const res = await api.get(`/api/organizations/${id}`)
  return res.data
}

export async function createOrganization(
  payload: OrganizationFormValues
): Promise<ApiResponse<Organization>> {
  const res = await api.post('/api/organizations/', payload)
  return res.data
}

export async function updateOrganization(
  payload: OrganizationFormValues & { id: number }
): Promise<ApiResponse<Organization>> {
  const res = await api.put('/api/organizations/', payload)
  return res.data
}

export async function deleteOrganization(
  id: number
): Promise<ApiResponse<null>> {
  const res = await api.delete(`/api/organizations/${id}`)
  return res.data
}
