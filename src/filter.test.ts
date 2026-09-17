import { describe, it, expect } from "vitest";
import { getBenchmarkVariables } from "./filter";
import { BenchmarkRun } from "./types";

// Sample data for testing - This data now aligns with the ActualBenchmarkRun type
const sampleRuns: BenchmarkRun[] = [
  {
    sourceFile: "test1.json",
    testName: "A",
    testDescription: "Desc A",
    outputDir: "/tmp/a",
    testConfig: { NodeType: "geth", GasLimit: 100, ExtraParam: "true" }, // Use boolean true
    result: { success: true },
  },
  {
    sourceFile: "test2.json",
    testName: "B",
    testDescription: "Desc B",
    outputDir: "/tmp/b",
    testConfig: { NodeType: "reth", GasLimit: 100, ExtraParam: "true" }, // Use boolean true
    result: { success: true },
  },
  {
    sourceFile: "test3.json",
    testName: "C",
    testDescription: "Desc C",
    outputDir: "/tmp/c",
    testConfig: { NodeType: "geth", GasLimit: 200, ExtraParam: "true" }, // Use boolean true
    result: { success: true },
  },
  {
    sourceFile: "test4.json",
    testName: "D",
    testDescription: "Desc D",
    outputDir: "/tmp/d",
    testConfig: { NodeType: "reth", GasLimit: 200, ExtraParam: "false" }, // Use boolean false
    result: { success: true },
  },
  {
    sourceFile: "test5.json",
    testName: "E",
    testDescription: "Desc E",
    outputDir: "/tmp/e",
    testConfig: { NodeType: "geth", GasLimit: 200, ExtraParam: "false" }, // Use boolean false
    result: { success: true },
  },
];

describe("BenchmarkVariables", () => {
  it("should update options and matched runs when a filter changes", () => {
    const result = getBenchmarkVariables(sampleRuns, {
      params: { GasLimit: 200, ExtraParam: "true" }, // Change GasLimit from default 100 to 200
      byMetric: "NodeType",
    });

    // Matched runs based on { GasLimit: 200, ExtraParam: true }
    expect(result.matchedRuns).toHaveLength(1);
    expect(result.matchedRuns[0].testName).toBe("C"); // geth, 200, true

    // Filter options available with GasLimit=200
    expect(result.filterOptions).toEqual({
      // GasLimit: [100, 200]
      // Check GasLimit=100 with other filters { ExtraParam: true }: run 'A' matches.
      // Check GasLimit=200 with other filters { ExtraParam: true }: run 'C' matches.
      GasLimit: [100, 200],
      // ExtraParam: [true, false]
      // Check ExtraParam=true with other filters { GasLimit: 200 }: run 'C' matches.
      // Check ExtraParam=false with other filters { GasLimit: 200 }: runs 'D', 'E' match.
      ExtraParam: ["false", "true"],
    });
  });

  it("should provide options even if the current selection has no matches", () => {
    const result = getBenchmarkVariables(sampleRuns, {
      params: { GasLimit: 100, ExtraParam: false }, // This specific combo has no runs
      byMetric: "NodeType",
    });

    // Active filters: { GasLimit: 100, ExtraParam: false }

    // Matched runs based on { GasLimit: 100, ExtraParam: false }
    expect(result.matchedRuns).toHaveLength(0);

    // Filter options should still be calculated based on *other* filters
    expect(result.filterOptions).toEqual({
      // GasLimit: [100, 200]
      // Check GasLimit=100 with other filter { ExtraParam: false }: No match.
      // Check GasLimit=200 with other filter { ExtraParam: false }: Runs 'D', 'E' match.
      GasLimit: [200],
      // ExtraParam: [true, false]
      // Check ExtraParam=true with other filter { GasLimit: 100 }: Run 'A', 'B' matches.
      // Check ExtraParam=false with other filter { GasLimit: 100 }: No match.
      ExtraParam: ["true"],
    });
  });

  it("should handle byMetric correctly, excluding it from filters and options", () => {
    const result = getBenchmarkVariables(sampleRuns, {
      params: { GasLimit: 200, NodeType: "geth" },
      byMetric: "ExtraParam", // Group by ExtraParam this time
    });

    // Active filters: { NodeType: 'geth', GasLimit: 200 } (default for NodeType)

    // Matched runs based on { NodeType: 'geth', GasLimit: 200 }
    expect(result.matchedRuns).toHaveLength(2);
    expect(result.matchedRuns.map((r) => r.testName).sort()).toEqual([
      "C",
      "E",
    ]); // geth, 200, true AND geth, 200, false

    // Filter options available (excluding byMetric 'ExtraParam')
    expect(result.filterOptions).toEqual({
      // NodeType: ['geth', 'reth']
      // Check NodeType='geth' with other { GasLimit: 200 }: Runs 'C', 'E' match.
      // Check NodeType='reth' with other { GasLimit: 200 }: Run 'D' matches.
      NodeType: ["geth", "reth"],
      // GasLimit: [100, 200]
      // Check GasLimit=100 with other { NodeType: 'geth' }: Run 'A' matches.
      // Check GasLimit=200 with other { NodeType: 'geth' }: Runs 'C', 'E' match.
      GasLimit: [100, 200],
    });
  });

  it("should handle scenario with only one variable", () => {
    const singleVarRuns: BenchmarkRun[] = [
      {
        sourceFile: "f1",
        testName: "T1",
        testDescription: "",
        outputDir: "",
        testConfig: { X: "a" },
        result: { success: true },
      },
      {
        sourceFile: "f2",
        testName: "T2",
        testDescription: "",
        outputDir: "",
        testConfig: { X: "b" },
        result: { success: true },
      },
    ];
    const result = getBenchmarkVariables(singleVarRuns, {
      params: {}, // No initial params
      byMetric: "X", // Group by the only variable
    });

    expect(result.variables).toEqual({ X: ["a", "b"] });
    // No active filters because the only variable is the byMetric
    expect(result.matchedRuns).toHaveLength(2); // All runs match initially
    // No filter options because the only variable is the byMetric
    expect(result.filterOptions).toEqual({});
  });

  it("should correctly filter options based on other selected filters, even with sparse data", () => {
    // Renamed test description for clarity
    // Runs where combinations don't overlap well for filtering
    const sparseRuns: BenchmarkRun[] = [
      {
        sourceFile: "s1",
        testName: "S1",
        testDescription: "",
        outputDir: "",
        testConfig: { P1: "A", P2: "X" },
        result: { success: true },
      },
      {
        sourceFile: "s2",
        testName: "S2",
        testDescription: "",
        outputDir: "",
        testConfig: { P1: "B", P2: "Y" },
        result: { success: true },
      },
      {
        sourceFile: "s3",
        testName: "S3",
        testDescription: "",
        outputDir: "",
        testConfig: { P1: "A", P2: "Z" },
        result: { success: true },
      }, // Added A/Z
    ];

    // Scenario 1: Select P1='A', group by P2
    const result1 = getBenchmarkVariables(sparseRuns, {
      params: { P1: "A" },
      byMetric: "P2",
    });

    expect(result1.variables).toEqual({ P1: ["A", "B"], P2: ["X", "Y", "Z"] });
    // Active filter: { P1: 'A' }
    expect(result1.matchedRuns).toHaveLength(2); // S1 (A,X), S3 (A,Z)
    expect(result1.matchedRuns.map((r) => r.testName).sort()).toEqual([
      "S1",
      "S3",
    ]);
    // Filter Options:
    // P1: Check P1='A' with other {}: Match S1, S3. Check P1='B' with other {}: Match S2.
    expect(result1.filterOptions).toEqual({ P1: ["A", "B"] });

    // Scenario 2: Select P1='B', group by P2
    const result2 = getBenchmarkVariables(sparseRuns, {
      params: { P1: "B" },
      byMetric: "P2",
    });
    // Active filter: { P1: 'B' }
    expect(result2.matchedRuns).toHaveLength(1); // S2 (B, Y)
    expect(result2.matchedRuns[0].testName).toBe("S2");
    // Filter Options:
    // P1: Check P1='A' with other {}: Match S1, S3. Check P1='B' with other {}: Match S2.
    expect(result2.filterOptions).toEqual({ P1: ["A", "B"] });
  });
});

describe("getBenchmarkVariables", () => {
  // Test for initial state (add this back if it was removed)
  it("should correctly identify variables, initial options, and matched runs with defaults", () => {
    const result = getBenchmarkVariables(sampleRuns, {
      params: {
        ExtraParam: "true",
        GasLimit: 100,
      },
      byMetric: "NodeType", // Group by NodeType
    });

    // Variables identified (more than one value)
    expect(result.variables).toEqual({
      NodeType: ["geth", "reth"],
      GasLimit: [100, 200],
      ExtraParam: ["false", "true"],
    });

    // Matched runs based on initial active filters
    expect(result.matchedRuns).toHaveLength(2);
    expect(result.matchedRuns.map((r) => r.testName).sort()).toEqual([
      "A",
      "B",
    ]); // geth, 200, false AND reth, 200, false

    // Filter options available initially (excluding byMetric)
    expect(result.filterOptions).toEqual({
      // GasLimit: Should check based on { ExtraParam: true }
      // -> GasLimit=100 matches A, B. GasLimit=200 matches C.
      GasLimit: [100, 200],
      // ExtraParam: Should check based on { GasLimit: 100 }
      // -> ExtraParam=true matches A, B. ExtraParam=false has no match with GasLimit=100.
      ExtraParam: ["true"], // Only true should be available based on default GasLimit=100
    });
  });
});
