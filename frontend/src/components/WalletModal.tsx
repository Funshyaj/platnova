import { useState } from "react";
import type { AccountView } from "../types";
import {
  useGetTransactionsQuery,
  useDepositMutation,
} from "../services/accountApi";

const PAGE_SIZE = 5;

export default function WalletModal({
  wallet,
  onClose,
}: {
  wallet: AccountView;
  onClose: () => void;
}) {
  const [page, setPage] = useState(1);
  const { data, isLoading, isError, error } = useGetTransactionsQuery({
    id: wallet.id,
    page,
    pageSize: PAGE_SIZE,
  });
  const [deposit, { isLoading: depositing, error: depositError }] =
    useDepositMutation();
  const [depositAmount, setDepositAmount] = useState("");

  const totalPages = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1;

  async function handleDeposit(e: React.FormEvent) {
    e.preventDefault();
    const amount = parseFloat(depositAmount);
    if (!amount || amount <= 0) return;
    try {
      await deposit({ id: wallet.id, amount }).unwrap();
      setDepositAmount("");
    } catch {
      // error surfaced via depositError below
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="max-h-[85vh] w-full max-w-xl overflow-y-auto rounded-lg bg-white p-6 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-start justify-between">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">
              {wallet.name}
            </h3>
            <p className="text-sm text-gray-500">
              {wallet.owner
                ? `${wallet.owner.name} · ${wallet.owner.email}`
                : "Platform account"}{" "}
              · {wallet.type}
            </p>
            <p className="mt-1 text-xl font-bold text-gray-900">
              {wallet.balance.toFixed(2)} {wallet.currency}
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
            aria-label="Close"
          >
            ✕
          </button>
        </div>

        <form className="mb-4 flex gap-2" onSubmit={handleDeposit}>
          <input
            type="number"
            step="0.01"
            min="0"
            placeholder="Deposit amount"
            value={depositAmount}
            onChange={(e) => setDepositAmount(e.target.value)}
            className="flex-1 rounded-md border border-gray-300 px-3 py-1.5 text-sm"
          />
          <button
            type="submit"
            disabled={depositing}
            className="rounded-md bg-blue-600 px-3 py-1.5 text-sm text-white disabled:opacity-60"
          >
            {depositing ? "Depositing..." : "Deposit"}
          </button>
        </form>
        {depositError && (
          <p className="mb-2 text-sm text-red-600">
            {JSON.stringify((depositError as any)?.data ?? depositError)}
          </p>
        )}

        <h4 className="mb-2 text-sm font-semibold text-gray-900">
          Transaction History
        </h4>
        {isLoading && (
          <p className="text-sm text-gray-500">Loading transactions...</p>
        )}
        {isError && (
          <p className="text-sm text-red-600">
            Failed to load transactions:{" "}
            {JSON.stringify((error as any)?.data ?? error)}
          </p>
        )}
        {data && data.data.length === 0 && (
          <p className="text-sm text-gray-500">No transactions yet.</p>
        )}

        {data && data.data.length > 0 && (
          <>
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr>
                  <th className="border-b border-gray-200 px-2 py-1.5 text-left">
                    Date
                  </th>
                  <th className="border-b border-gray-200 px-2 py-1.5 text-left">
                    Type
                  </th>
                  <th className="border-b border-gray-200 px-2 py-1.5 text-left">
                    Amount
                  </th>
                  <th className="border-b border-gray-200 px-2 py-1.5 text-left">
                    Balance After
                  </th>
                </tr>
              </thead>
              <tbody>
                {data.data.map((t) => (
                  <tr key={t.id}>
                    <td className="border-b border-gray-200 px-2 py-1.5">
                      {new Date(t.created_at).toLocaleString()}
                    </td>
                    <td className="border-b border-gray-200 px-2 py-1.5">
                      {t.type}
                    </td>
                    <td className="border-b border-gray-200 px-2 py-1.5">
                      {t.type === "transfer_out" ? "-" : "+"}
                      {t.amount.toFixed(2)} {t.currency}
                    </td>
                    <td className="border-b border-gray-200 px-2 py-1.5">
                      {t.balance_after.toFixed(2)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="mt-3 flex items-center gap-3 text-sm">
              <button
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
                className="rounded-md border border-gray-300 px-2 py-1 disabled:opacity-40"
              >
                Prev
              </button>
              <span>
                Page {page} of {totalPages}
              </span>
              <button
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
                className="rounded-md border border-gray-300 px-2 py-1 disabled:opacity-40"
              >
                Next
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
