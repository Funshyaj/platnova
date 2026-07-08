import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

// Shared RTK Query instance. accountApi.ts and transferApi.ts inject their
// own endpoints into this via injectEndpoints so they share one cache/tag
// space while staying in separate files per resource.
export const baseApi = createApi({
  reducerPath: 'api',
  baseQuery: fetchBaseQuery({ baseUrl: BASE_URL }),
  tagTypes: ['Account', 'Transactions'],
  endpoints: () => ({}),
})
