import { describe, expect, it } from "vitest";

import { formatTransactions } from "./ConfigCard";

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
