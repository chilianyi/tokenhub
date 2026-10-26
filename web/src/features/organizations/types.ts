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
import { z } from 'zod'

// ============================================================================
// Organization Schema & Types
//
// 部门额度语义（分配池）：
//   quota      部门充值进来的总额度池
//   allocated  已分配给部门成员的额度总和
//   不变式     allocated <= quota
// ============================================================================

export const organizationSchema = z.object({
  id: z.number(),
  name: z.string(),
  quota: z.number(),
  allocated: z.number(),
  status: z.number(), // 1: enabled, 0: disabled
  remark: z.string().optional().default(''),
  created_time: z.number(),
  updated_time: z.number(),
})

export type Organization = z.infer<typeof organizationSchema>

export interface OrganizationFormValues {
  name: string
  quota: number
  status: number
  remark: string
}

export interface ApiResponse<T> {
  success: boolean
  message: string
  data: T
}

/** 后端 common.PageInfo 的分页信封 */
export interface PaginatedOrganizations {
  page: number
  page_size: number
  total: number
  items: Organization[]
}

export interface SearchOrganizationsParams {
  keyword?: string
  p?: number
  page_size?: number
}
