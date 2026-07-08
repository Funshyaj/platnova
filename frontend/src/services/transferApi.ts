import { baseApi } from './baseApi'
import type { TransferRequest, TransferResult } from '../types'

export const transferApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    transfer: builder.mutation<TransferResult, TransferRequest>({
      query: (body) => ({
        url: '/transfers',
        method: 'POST',
        headers: { 'Idempotency-Key': crypto.randomUUID() },
        body,
      }),
      invalidatesTags: (result) =>
        result
          ? [
              { type: 'Account', id: 'LIST' },
              { type: 'Account', id: 'STATS' },
              { type: 'Account', id: result.from_account.id },
              { type: 'Account', id: result.to_account.id },
              { type: 'Transactions', id: result.from_account.id },
              { type: 'Transactions', id: result.to_account.id },
            ]
          : [{ type: 'Account', id: 'LIST' }],
    }),
  }),
})

export const { useTransferMutation } = transferApi
