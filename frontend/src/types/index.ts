export interface User {
  id: string
  name: string
  email: string
  created_at: string
}

export interface Currency {
  code: string
  full_name: string
  symbol: string
}

export interface Account {
  id: string
  user_id?: string
  name: string
  type: string
  currency: string
  balance: number
  created_at: string
}

export interface AccountView extends Account {
  owner?: User
}

export interface Transaction {
  id: string
  account_id: string
  type: 'deposit' | 'transfer_in' | 'transfer_out'
  amount: number
  currency: string
  related_account_id?: string
  transfer_id?: string
  rate?: number
  balance_after: number
  created_at: string
}

export interface TransactionsResponse {
  data: Transaction[]
  page: number
  page_size: number
  total: number
}

export interface Stats {
  total_accounts: number
  total_transfers: number
  total_balance_usd: number
  vault_balance_usd: number
}

export interface Rate {
  pair: string
  base: string
  quote: string
  rate: number
}

export interface TransferRequest {
  from_account_id: string
  to_account_id: string
  amount: number
}

export interface TransferResult {
  transfer_id: string
  from: Transaction
  to: Transaction
  rate: number
  converted_amount: number
  from_account: AccountView
  to_account: AccountView
}
