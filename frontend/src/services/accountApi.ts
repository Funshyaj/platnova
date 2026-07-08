import { baseApi } from './baseApi'
import type { AccountView, Transaction, TransactionsResponse, Stats, User, Currency, Rate } from '../types'

export const accountApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    getAccounts: builder.query<AccountView[], void>({
      query: () => '/accounts',
      providesTags: (result) =>
        result
          ? [...result.map((a) => ({ type: 'Account' as const, id: a.id })), { type: 'Account', id: 'LIST' }]
          : [{ type: 'Account', id: 'LIST' }],
    }),
    getAccount: builder.query<AccountView, string>({
      query: (id) => `/accounts/${id}`,
      providesTags: (_result, _error, id) => [{ type: 'Account', id }],
    }),
    getTransactions: builder.query<TransactionsResponse, { id: string; page?: number; pageSize?: number }>({
      query: ({ id, page = 1, pageSize = 10 }) => `/accounts/${id}/transactions?page=${page}&page_size=${pageSize}`,
      providesTags: (_result, _error, arg) => [{ type: 'Transactions', id: arg.id }],
    }),
    getRates: builder.query<Rate[], void>({
      query: () => '/rates',
    }),
    getStats: builder.query<Stats, void>({
      query: () => '/stats',
      providesTags: [{ type: 'Account', id: 'STATS' }],
    }),
    getUsers: builder.query<User[], void>({
      query: () => '/users',
    }),
    getCurrencies: builder.query<Currency[], void>({
      query: () => '/currencies',
    }),
    deposit: builder.mutation<{ account: AccountView; transaction: Transaction }, { id: string; amount: number }>({
      query: ({ id, amount }) => ({
        url: `/accounts/${id}/deposit`,
        method: 'POST',
        body: { amount },
      }),
      invalidatesTags: (_result, _error, arg) => [
        { type: 'Account', id: arg.id },
        { type: 'Account', id: 'LIST' },
        { type: 'Account', id: 'STATS' },
        { type: 'Transactions', id: arg.id },
      ],
    }),
    createAccount: builder.mutation<AccountView, { name: string; currency: string; user_id?: string }>({
      query: (body) => ({
        url: '/accounts',
        method: 'POST',
        body,
      }),
      invalidatesTags: [{ type: 'Account', id: 'LIST' }],
    }),
  }),
})

export const {
  useGetAccountsQuery,
  useGetAccountQuery,
  useGetTransactionsQuery,
  useGetRatesQuery,
  useGetStatsQuery,
  useGetUsersQuery,
  useGetCurrenciesQuery,
  useDepositMutation,
  useCreateAccountMutation,
} = accountApi
