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
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import {
  SideDrawerSection,
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { handleServerError } from '@/lib/handle-server-error'

import { createOrganization, updateOrganization } from '../api'
import { useOrganizations } from './organizations-provider'

// 表单内部用 string 承载额度输入（Input 的值天然是字符串），
// 提交时再转成 number，避免 z.coerce 造成的 input/output 类型分裂。
const schema = z.object({
  name: z.string().trim().min(1).max(128),
  quota: z.string().regex(/^\d+$/),
  remark: z.string().max(255),
  status: z.number().int(),
})

type FormValues = z.infer<typeof schema>

export function OrganizationsMutateDrawer({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const { currentRow, triggerRefresh } = useOrganizations()
  const isUpdate = !!currentRow
  const [isSubmitting, setIsSubmitting] = useState(false)

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: '', quota: '0', remark: '', status: 1 },
  })

  useEffect(() => {
    if (!open) return
    if (currentRow) {
      form.reset({
        name: currentRow.name,
        quota: String(currentRow.quota),
        remark: currentRow.remark || '',
        status: currentRow.status,
      })
    } else {
      form.reset({ name: '', quota: '0', remark: '', status: 1 })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, currentRow?.id])

  const onSubmit = async (values: FormValues) => {
    if (isUpdate && !currentRow) return
    setIsSubmitting(true)
    try {
      const payload = {
        name: values.name,
        quota: Number(values.quota),
        remark: values.remark,
        status: values.status,
      }
      const result =
        isUpdate && currentRow
          ? await updateOrganization({ ...payload, id: currentRow.id })
          : await createOrganization(payload)
      if (result.success) {
        toast.success(
          isUpdate ? t('Organization updated') : t('Organization created')
        )
        onOpenChange(false)
        triggerRefresh()
      } else {
        handleServerError(result)
      }
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsSubmitting(false)
    }
  }

  const allocated = currentRow?.allocated ?? 0

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[520px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>
            {isUpdate ? t('Update Organization') : t('Create Organization')}
          </SheetTitle>
          <SheetDescription>
            {t(
              'The quota pool caps the total amount that can be allocated to members of this organization.'
            )}
          </SheetDescription>
        </SheetHeader>

        <Form {...form}>
          <form
            id='organization-form'
            onSubmit={form.handleSubmit(onSubmit)}
            className={sideDrawerFormClassName()}
          >
            <fieldset disabled={isSubmitting} className='contents'>
              <SideDrawerSection>
                <FormField
                  control={form.control}
                  name='name'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Name')}</FormLabel>
                      <FormControl>
                        <Input {...field} placeholder={t('e.g. AI Research')} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='quota'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Quota Pool')}</FormLabel>
                      <FormControl>
                        <Input {...field} type='number' min={0} step={1} />
                      </FormControl>
                      <FormDescription>
                        {isUpdate
                          ? t(
                              'Cannot be set below the amount already allocated ({{allocated}}).',
                              { allocated }
                            )
                          : t(
                              'Members can be allocated at most this much in total.'
                            )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='remark'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Remark')}</FormLabel>
                      <FormControl>
                        <Input {...field} placeholder={t('Optional')} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </SideDrawerSection>
            </fieldset>
          </form>
        </Form>

        <SheetFooter className={sideDrawerFooterClassName()}>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={isSubmitting}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='submit'
            form='organization-form'
            disabled={isSubmitting}
          >
            {isSubmitting ? t('Saving...') : t('Save')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
