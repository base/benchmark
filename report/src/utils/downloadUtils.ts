import { BenchmarkRunWithStatus } from "../types";

/**
 * Downloads run information as a JSON file
 */
export const downloadRunInfo = (
  runs: BenchmarkRunWithStatus[],
  gasConfigName: string,
): void => {
  // Create a comprehensive data structure with all raw run details
  const downloadData = {
    gasConfiguration: gasConfigName,
    downloadedAt: new Date().toISOString(),
    totalRuns: runs.length,
    runs: runs.map((run) => ({
      // Basic run information
      id: run.id,
      sourceFile: run.sourceFile,
      testName: run.testName,
      testDescription: run.testDescription,
      outputDir: run.outputDir,
      bucketPath: run.bucketPath,
      createdAt: run.createdAt,
      status: run.status,
      
      // Test configuration
      testConfig: run.testConfig,
      
      // Machine information
      machineInfo: run.machineInfo || null,
      
      // Performance results (raw data)
      result: run.result,
      
      // Thresholds
      thresholds: run.thresholds || null,
    })),
  };

  // Create and download the file
  const blob = new Blob([JSON.stringify(downloadData, null, 2)], {
    type: "application/json",
  });
  
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  
  // Create a safe filename
  const safeGasConfigName = gasConfigName.replace(/[^a-z0-9]/gi, '_').toLowerCase();
  const timestamp = new Date().toISOString().split('T')[0]; // YYYY-MM-DD format
  link.download = `benchmark_run_${safeGasConfigName}_${timestamp}.json`;
  
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
};


/**
 * Downloads run information as a CSV file
 */
export const downloadRunInfoCSV = (
  runs: BenchmarkRunWithStatus[],
  gasConfigName: string,
): void => {
  // Define CSV headers
  const headers = [
    'Run ID',
    'Test Name',
    'Output Dir',
    'Status',
    'Created At',
    'Gas Limit',
    'Block Time (ms)',
    'Node Type',
    'Transaction Payload',
    'Client Version',
    'Machine Type',
    'Machine Provider',
    'Machine Region',
    'Machine File System',
    'Sequencer Gas/s',
    'Fork Choice Updated (s)',
    'Get Payload (s)',
    'Send Txs (s)',
    'Validator Gas/s',
    'New Payload (s)',
    'Success',
    'Complete'
  ];

  // Convert runs to CSV rows
  const csvRows = runs.map(run => [
    run.id,
    run.testName,
    run.outputDir,
    run.status,
    run.createdAt,
    run.testConfig.GasLimit || '',
    run.testConfig.BlockTimeMilliseconds || '',
    run.testConfig.NodeType || '',
    run.testConfig.TransactionPayload || '',
    run.result?.clientVersion || '',
    run.machineInfo?.type || '',
    run.machineInfo?.provider || '',
    run.machineInfo?.region || '',
    run.machineInfo?.fileSystem || '',
    run.result?.sequencerMetrics?.gasPerSecond || '',
    run.result?.sequencerMetrics?.forkChoiceUpdated || '',
    run.result?.sequencerMetrics?.getPayload || '',
    run.result?.sequencerMetrics?.sendTxs || '',
    run.result?.validatorMetrics?.gasPerSecond || '',
    run.result?.validatorMetrics?.newPayload || '',
    run.result?.success || false,
    run.result?.complete || false
  ]);

  // Combine headers and rows
  const csvContent = [headers, ...csvRows]
    .map(row => row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(','))
    .join('\n');

  // Create and download the file
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  
  // Create a safe filename
  const safeGasConfigName = gasConfigName.replace(/[^a-z0-9]/gi, '_').toLowerCase();
  const timestamp = new Date().toISOString().split('T')[0]; // YYYY-MM-DD format
  link.download = `benchmark_run_${safeGasConfigName}_${timestamp}.csv`;
  
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
};

/**
 * Downloads all run information as a JSON file
 */
export const downloadAllRuns = (
  allRuns: BenchmarkRunWithStatus[],
  testName: string,
): void => {
  // Group runs by gas configuration for better organization
  const runsByGasConfig = allRuns.reduce((acc, run) => {
    const gasLimit = Number(run.testConfig?.GasLimit) || 0;
    const blockTimeMs = Number(run.testConfig?.BlockTimeMilliseconds) || 2000;
    const gasPerSecond = gasLimit / (blockTimeMs / 1000);
    const gasConfigKey = `${gasPerSecond.toFixed(0)}_gas_per_second`;
    
    if (!acc[gasConfigKey]) {
      acc[gasConfigKey] = {
        gasPerSecond,
        gasLimit,
        blockTimeMs,
        runs: []
      };
    }
    acc[gasConfigKey].runs.push(run);
    return acc;
  }, {} as Record<string, { gasPerSecond: number; gasLimit: number; blockTimeMs: number; runs: BenchmarkRunWithStatus[] }>);

  // Create comprehensive data structure with all runs organized by gas configuration
  const downloadData = {
    testName,
    downloadedAt: new Date().toISOString(),
    totalRuns: allRuns.length,
    gasConfigurations: Object.entries(runsByGasConfig).map(([, config]) => ({
      gasPerSecond: config.gasPerSecond,
      gasLimit: config.gasLimit,
      blockTimeMilliseconds: config.blockTimeMs,
      runCount: config.runs.length,
      runs: config.runs.map((run) => ({
        // Basic run information
        id: run.id,
        sourceFile: run.sourceFile,
        testName: run.testName,
        testDescription: run.testDescription,
        outputDir: run.outputDir,
        bucketPath: run.bucketPath,
        createdAt: run.createdAt,
        status: run.status,
        
        // Test configuration
        testConfig: run.testConfig,
        
        // Machine information
        machineInfo: run.machineInfo || null,
        
        // Performance results (raw data)
        result: run.result,
        
        // Thresholds
        thresholds: run.thresholds || null,
      }))
    })),
    
    // Overall statistics (counts only, no averages)
    summary: {
      totalGasConfigurations: Object.keys(runsByGasConfig).length,
      statusCounts: allRuns.reduce((acc, run) => {
        acc[run.status] = (acc[run.status] || 0) + 1;
        return acc;
      }, {} as Record<string, number>),
    }
  };

  // Create and download the file
  const blob = new Blob([JSON.stringify(downloadData, null, 2)], {
    type: "application/json",
  });
  
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  
  // Create a safe filename
  const safeTestName = testName.replace(/[^a-z0-9]/gi, '_').toLowerCase();
  const timestamp = new Date().toISOString().split('T')[0]; // YYYY-MM-DD format
  link.download = `benchmark_all_runs_${safeTestName}_${timestamp}.json`;
  
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
};

/**
 * Downloads all run information as a CSV file
 */
export const downloadAllRunsCSV = (
  allRuns: BenchmarkRunWithStatus[],
  testName: string,
): void => {
  // Define CSV headers (same as individual download)
  const headers = [
    'Run ID',
    'Test Name',
    'Output Dir',
    'Status',
    'Created At',
    'Gas Limit',
    'Block Time (ms)',
    'Node Type',
    'Transaction Payload',
    'Client Version',
    'Machine Type',
    'Machine Provider',
    'Machine Region',
    'Machine File System',
    'Sequencer Gas/s',
    'Fork Choice Updated (s)',
    'Get Payload (s)',
    'Send Txs (s)',
    'Validator Gas/s',
    'New Payload (s)',
    'Success',
    'Complete'
  ];

  // Convert all runs to CSV rows
  const csvRows = allRuns.map(run => [
    run.id,
    run.testName,
    run.outputDir,
    run.status,
    run.createdAt,
    run.testConfig.GasLimit || '',
    run.testConfig.BlockTimeMilliseconds || '',
    run.testConfig.NodeType || '',
    run.testConfig.TransactionPayload || '',
    run.result?.clientVersion || '',
    run.machineInfo?.type || '',
    run.machineInfo?.provider || '',
    run.machineInfo?.region || '',
    run.machineInfo?.fileSystem || '',
    run.result?.sequencerMetrics?.gasPerSecond || '',
    run.result?.sequencerMetrics?.forkChoiceUpdated || '',
    run.result?.sequencerMetrics?.getPayload || '',
    run.result?.sequencerMetrics?.sendTxs || '',
    run.result?.validatorMetrics?.gasPerSecond || '',
    run.result?.validatorMetrics?.newPayload || '',
    run.result?.success || false,
    run.result?.complete || false
  ]);

  // Combine headers and rows
  const csvContent = [headers, ...csvRows]
    .map(row => row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(','))
    .join('\n');

  // Create and download the file
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  
  // Create a safe filename
  const safeTestName = testName.replace(/[^a-z0-9]/gi, '_').toLowerCase();
  const timestamp = new Date().toISOString().split('T')[0]; // YYYY-MM-DD format
  link.download = `benchmark_all_runs_${safeTestName}_${timestamp}.csv`;
  
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
};
