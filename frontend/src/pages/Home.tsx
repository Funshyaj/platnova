import { useMemo, useState } from "react";
import {
  useGetAccountsQuery,
  useGetStatsQuery,
  useGetCurrenciesQuery,
} from "../services/accountApi";
import type { AccountView } from "../types";
import MetricCard from "../components/MetricCard";
import WalletModal from "../components/WalletModal";

export default function Home() {
  const {
    data: accounts,
    isLoading,
    isError,
    error,
    refetch,
  } = useGetAccountsQuery();
  const { data: stats } = useGetStatsQuery();
  const { data: currencies } = useGetCurrenciesQuery();
  const [activeCurrency, setActiveCurrency] = useState<string | null>(null);
  const [selectedWallet, setSelectedWallet] = useState<AccountView | null>(
    null,
  );

  const wallets = useMemo(
    () => (accounts ?? []).filter((a) => a.type !== "Vault"),
    [accounts],
  );

  const tabs = useMemo(() => {
    if (currencies && currencies.length > 0)
      return currencies.map((c) => c.code);
    return Array.from(new Set(wallets.map((w) => w.currency)));
  }, [currencies, wallets]);

  const currentTab = activeCurrency ?? tabs[0] ?? null;
  const walletsForTab = wallets.filter((w) => w.currency === currentTab);

  if (isLoading)
    return <p className="text-sm text-gray-500">Loading dashboard...</p>;
  if (isError) {
    return (
      <div className="text-sm text-red-600">
        <p>
          Failed to load accounts:{" "}
          {JSON.stringify((error as any)?.data ?? error)}
        </p>
        <button
          onClick={() => refetch()}
          className="mt-2 rounded-md border border-gray-300 px-3 py-1"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-8 grid grid-cols-4 gap-4">
        <MetricCard
          label="Total Wallet Accounts"
          value={stats ? stats.total_accounts.toString() : "—"}
        />
        <MetricCard
          label="Accounts Balance (USD equiv.)"
          value={stats ? `$${stats.total_balance_usd.toFixed(2)}` : "—"}
        />
        <MetricCard
          label="Transfers"
          value={stats ? stats.total_transfers.toString() : "—"}
        />
        <MetricCard
          label="Vault Balance"
          value={stats ? `$${stats.vault_balance_usd.toFixed(2)}` : "—"}
        />
      </div>

      <h2 className="mb-2 text-lg font-semibold text-gray-900">Wallets</h2>

      <div className="mb-4 flex gap-1 border-b border-gray-200">
        {tabs.map((code) => (
          <button
            key={code}
            onClick={() => setActiveCurrency(code)}
            className={
              code === currentTab
                ? "border-b-2 border-blue-600 px-4 py-2 text-sm font-medium text-blue-600"
                : "border-b-2 border-transparent px-4 py-2 text-sm font-medium text-gray-500 hover:text-gray-700"
            }
          >
            {code}
          </button>
        ))}
      </div>

      {walletsForTab.length === 0 ? (
        <p className="text-sm text-gray-500">No {currentTab} wallets yet.</p>
      ) : (
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr>
              <th className="border-b border-gray-200 px-2 py-2 text-left">
                User
              </th>
              <th className="border-b border-gray-200 px-2 py-2 text-left">
                Email
              </th>
              <th className="border-b border-gray-200 px-2 py-2 text-left">
                Type
              </th>
              <th className="border-b border-gray-200 px-2 py-2 text-left">
                Balance
              </th>
            </tr>
          </thead>
          <tbody>
            {walletsForTab.map((w) => (
              <tr
                key={w.id}
                onClick={() => setSelectedWallet(w)}
                className="cursor-pointer hover:bg-gray-50"
              >
                <td className="border-b border-gray-200 px-2 py-2">
                  {w.owner?.name ?? "—"}
                </td>
                <td className="border-b border-gray-200 px-2 py-2">
                  {w.owner?.email ?? "—"}
                </td>
                <td className="border-b border-gray-200 px-2 py-2">{w.type}</td>
                <td className="border-b border-gray-200 px-2 py-2">
                  {w.balance.toFixed(2)} {w.currency}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {selectedWallet && (
        <WalletModal
          wallet={selectedWallet}
          onClose={() => setSelectedWallet(null)}
        />
      )}
    </div>
  );
}
