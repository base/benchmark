import { describe, expect, it } from "vitest";

import { buildRows, formatTransactions } from "./ConfigCard";
import { LoadTestConfig } from "../types";

const baseConfig = (
  overrides: Partial<LoadTestConfig> = {},
): LoadTestConfig => ({
  funding_amount: "1000000000000000000",
  sender_count: 10,
  sender_offset: 0,
  in_flight_per_sender: 4,
  batch_size: 50,
  batch_timeout: "100ms",
  duration: "60s",
  target_gps: 1_000_000,
  seed: 42,
  chain_id: 84532,
  transactions: [{ type: "swap", weight: 100 }],
  looper_contract: null,
  swap_token_amount: "0",
  ...overrides,
});

const flatLabels = (config: LoadTestConfig): string[] =>
  buildRows(config)
    .flat()
    .map((r) => r.label);

describe("formatTransactions", () => {
  it("formats a weighted transaction mix", () => {
    expect(
      formatTransactions({
        transactions: [
          { type: "uniswap_v3", weight: 50 },
          { type: "aerodrome_cl", weight: 50 },
        ],
      }),
    ).toBe("uniswap_v3 (50%) · aerodrome_cl (50%)");
  });

  it("labels full fresh-recipient transfers as account-create", () => {
    expect(
      formatTransactions({
        fresh_recipient_ratio: 1,
        transactions: [{ type: "transfer", weight: 100 }],
      }),
    ).toBe("account-create (100%)");
  });

  it("keeps partial fresh-recipient transfer ratios visible", () => {
    expect(
      formatTransactions({
        fresh_recipient_ratio: 0.25,
        transactions: [{ type: "transfer", weight: 100 }],
      }),
    ).toBe("transfer (100%, 25% account-create)");
  });
});

describe("buildRows", () => {
  it("renders the full closed-loop config when all Option fields are set", () => {
    const labels = flatLabels(baseConfig());
    expect(labels).toEqual([
      "Senders",
      "In-flight / sender",
      "Batch size",
      "Batch timeout",
      "Duration",
      "Target gas/s",
      "Funding / sender",
      "Seed",
      "Chain ID",
    ]);
  });

  it("omits null Option fields instead of calling toLocaleString on them", () => {
    const labels = flatLabels(
      baseConfig({
        batch_timeout: null,
        duration: null,
        target_gps: null,
        chain_id: null,
      }),
    );
    expect(labels).toEqual([
      "Senders",
      "In-flight / sender",
      "Batch size",
      "Funding / sender",
      "Seed",
    ]);
    expect(labels).not.toContain("Target gas/s");
    expect(labels).not.toContain("Duration");
    expect(labels).not.toContain("Batch timeout");
    expect(labels).not.toContain("Chain ID");
  });

  it("omits Option fields that are missing (undefined) on older payloads", () => {
    // Simulate a JSON payload where keys were absent rather than null.
    const sparse = baseConfig();
    delete (sparse as { batch_timeout?: string | null }).batch_timeout;
    delete (sparse as { duration?: string | null }).duration;
    delete (sparse as { target_gps?: number | null }).target_gps;
    delete (sparse as { chain_id?: number | null }).chain_id;
    delete (sparse as { sender_offset?: number }).sender_offset;

    expect(() => buildRows(sparse)).not.toThrow();
    const labels = flatLabels(sparse);
    expect(labels).not.toContain("Target gas/s");
    expect(labels).not.toContain("Sender offset");
    expect(labels).not.toContain("Chain ID");
  });

  it("formats target_gps with SI suffixes when present", () => {
    const rows = buildRows(baseConfig({ target_gps: 25_000_000 })).flat();
    const target = rows.find((r) => r.label === "Target gas/s");
    expect(target?.value).toBe("25M gas/s");
  });

  it("renders the open-loop sepolia payload without batch_size (live crash repro)", () => {
    // Shape from https://base-benchmarking-api-dev.cbhq.net/api/v1/load-tests/sepolia/2026-07-27T00-01-29
    // — producer dropped batch_size/batch_timeout and emits funding_batch_size.
    const openLoop = baseConfig({
      sender_count: 300,
      in_flight_per_sender: 40,
      duration: "30s",
      target_gps: 375_000_000,
      seed: 234521,
      chain_id: null,
      funding_batch_size: 16,
      funding_amount: "20000000000000000",
      swap_token_amount: "1000000000000000000000",
      transactions: [
        { type: "uniswap_v3", weight: 50 },
        { type: "aerodrome_cl", weight: 50 },
      ],
    });
    delete (openLoop as { batch_size?: number | null }).batch_size;
    delete (openLoop as { batch_timeout?: string | null }).batch_timeout;

    expect(() => buildRows(openLoop)).not.toThrow();
    const labels = flatLabels(openLoop);
    expect(labels).toEqual([
      "Senders",
      "In-flight / sender",
      "Duration",
      "Target gas/s",
      "Funding / sender",
      "Funding batch size",
      "Seed",
    ]);
    expect(labels).not.toContain("Batch size");
    expect(labels).not.toContain("Batch timeout");
  });

  it("falls back to max_target_gps when target_gps is absent", () => {
    const rows = buildRows(
      baseConfig({ target_gps: undefined, max_target_gps: 50_000_000 }),
    ).flat();
    const target = rows.find((r) => r.label === "Target gas/s");
    expect(target?.value).toBe("50M gas/s");
  });
});
