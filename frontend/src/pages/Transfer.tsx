import { useMemo, useState } from "react";
import {
  useGetAccountsQuery,
  useGetUsersQuery,
  useGetRatesQuery,
} from "../services/accountApi";
import { useTransferMutation } from "../services/transferApi";
import type { Rate } from "../types";

function ratesToMap(rates: Rate[]): Record<string, number> {
  const map: Record<string, number> = {};
  for (const r of rates) map[r.pair] = r.rate;
  return map;
}

function convert(
  amount: number,
  from: string,
  to: string,
  rateMap: Record<string, number>,
): number | null {
  if (from === to) return amount;
  const usdPer = (ccy: string) =>
    ccy === "USD"
      ? 1
      : rateMap[`USD_${ccy}`]
        ? 1 / rateMap[`USD_${ccy}`]
        : null;
  const fromUsd = usdPer(from);
  const toUsd = usdPer(to);
  if (fromUsd === null || toUsd === null) return null;
  return (amount * fromUsd) / toUsd;
}

function PartySelector({
  label,
  users,
  wallets,
  userId,
  onUserChange,
  walletId,
  onWalletChange,
}: {
  label: string;
  users: { id: string; name: string; email: string }[];
  wallets: {
    id: string;
    currency: string;
    balance: number;
    user_id?: string;
  }[];
  userId: string;
  onUserChange: (id: string) => void;
  walletId: string;
  onWalletChange: (id: string) => void;
}) {
  const userWallets = wallets.filter((w) => w.user_id === userId);
  const selectedWallet = userWallets.find((w) => w.id === walletId);

  return (
    <fieldset className="rounded-md border border-gray-200 p-3">
      <legend className="px-1 text-sm font-medium text-gray-700">
        {label}
      </legend>
      <div className="flex flex-col gap-3">
        <label className="flex flex-col gap-1 text-sm text-gray-700">
          User
          <select
            value={userId}
            onChange={(e) => {
              onUserChange(e.target.value);
              onWalletChange("");
            }}
            className="rounded-md border border-gray-300 px-3 py-2 text-sm"
          >
            <option value="">Select user</option>
            {users.map((u) => (
              <option key={u.id} value={u.id}>
                {u.name} ({u.email})
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1 text-sm text-gray-700">
          Wallet
          <select
            value={walletId}
            onChange={(e) => onWalletChange(e.target.value)}
            disabled={!userId}
            className="rounded-md border border-gray-300 px-3 py-2 text-sm disabled:bg-gray-50"
          >
            <option value="">Select wallet</option>
            {userWallets.map((w) => (
              <option key={w.id} value={w.id}>
                {w.currency} — {w.balance.toFixed(2)}
              </option>
            ))}
          </select>
        </label>

        {selectedWallet && (
          <p className="text-sm text-gray-500">
            Balance: {selectedWallet.balance.toFixed(2)}{" "}
            {selectedWallet.currency}
          </p>
        )}
      </div>
    </fieldset>
  );
}

export default function Transfer() {
  const { data: accounts, isLoading: accountsLoading } = useGetAccountsQuery();
  const { data: users, isLoading: usersLoading } = useGetUsersQuery();
  const { data: rates, isLoading: ratesLoading } = useGetRatesQuery();
  const [transfer, { isLoading: submitting }] = useTransferMutation();

  const [fromUserId, setFromUserId] = useState("");
  const [fromWalletId, setFromWalletId] = useState("");
  const [toUserId, setToUserId] = useState("");
  const [toWalletId, setToWalletId] = useState("");
  const [amount, setAmount] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [apiError, setApiError] = useState<string | null>(null);

  const wallets = accounts ?? [];
  const rateMap = useMemo(() => (rates ? ratesToMap(rates) : {}), [rates]);

  const fromWallet = wallets.find((w) => w.id === fromWalletId);
  const toWallet = wallets.find((w) => w.id === toWalletId);

  const preview = useMemo(() => {
    const numericAmount = parseFloat(amount);
    if (!fromWallet || !toWallet || !numericAmount || numericAmount <= 0)
      return null;
    return convert(
      numericAmount,
      fromWallet.currency,
      toWallet.currency,
      rateMap,
    );
  }, [amount, fromWallet, toWallet, rateMap]);

  function validate(): string | null {
    const numericAmount = parseFloat(amount);
    if (!fromWalletId || !toWalletId)
      return "Select both a sender and receiver wallet.";
    if (fromWalletId === toWalletId)
      return "Source and destination wallets must be different.";
    if (!numericAmount || numericAmount <= 0)
      return "Enter an amount greater than zero.";
    if (fromWallet && numericAmount > fromWallet.balance)
      return "Insufficient funds in source wallet.";
    return null;
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSuccessMsg(null);
    setApiError(null);
    const validationError = validate();
    setFormError(validationError);
    if (validationError) return;

    try {
      const result = await transfer({
        from_account_id: fromWalletId,
        to_account_id: toWalletId,
        amount: parseFloat(amount),
      }).unwrap();
      setSuccessMsg(
        `Transfer complete: ${result.from.amount.toFixed(2)} ${result.from.currency} -> ${result.to.amount.toFixed(2)} ${result.to.currency} (rate ${result.rate.toFixed(4)})`,
      );
      setAmount("");
    } catch (err: any) {
      setApiError(err?.data?.error ?? "Transfer failed. Please try again.");
    }
  }

  if (accountsLoading || usersLoading || ratesLoading)
    return <p className="text-sm text-gray-500">Loading...</p>;

  return (
    <div className="space-y-3">
      <h2 className="text-xl font-semibold text-gray-900">New Transfer</h2>
      <div className="flex gap-8">
        <form
          onSubmit={handleSubmit}
          className="basis-full max-w-2xl flex flex-col gap-4"
        >
          <PartySelector
            label="From"
            users={users ?? []}
            wallets={wallets}
            userId={fromUserId}
            onUserChange={setFromUserId}
            walletId={fromWalletId}
            onWalletChange={setFromWalletId}
          />
          <PartySelector
            label="To"
            users={users ?? []}
            wallets={wallets}
            userId={toUserId}
            onUserChange={setToUserId}
            walletId={toWalletId}
            onWalletChange={setToWalletId}
          />
          <label className="flex flex-col gap-1 text-sm text-gray-700">
            Amount
            <input
              type="number"
              step="0.01"
              min="0"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder={
                fromWallet ? `Amount in ${fromWallet.currency}` : "Amount"
              }
              className="rounded-md border border-gray-300 px-3 py-2 text-sm"
            />
          </label>
          {preview !== null &&
            fromWallet &&
            toWallet &&
            fromWallet.currency !== toWallet.currency && (
              <p className="text-lg font-semibold text-blue-600">
                ≈ {preview.toFixed(2)} {toWallet.currency}
              </p>
            )}
          {formError && <p className="text-sm text-red-600">{formError}</p>}
          {apiError && <p className="text-sm text-red-600">{apiError}</p>}
          {successMsg && <p className="text-sm text-green-600">{successMsg}</p>}
          <button
            type="submit"
            disabled={submitting}
            className="self-start rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
          >
            {submitting ? "Sending..." : "Send Transfer"}
          </button>
        </form>
        {rates && (
          <div className="text-sm text-gray-600">
            <h3 className="mb-2 font-semibold text-gray-900">
              Static FX Rates (per 1 USD)
            </h3>
            <ul className="space-y-1">
              {rates.map((r) => (
                <li key={r.pair}>
                  {r.base} → {r.quote}: {r.rate}
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}
