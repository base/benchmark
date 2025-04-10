import { ChartConfig } from './types'; // Import from types.ts

export const CHART_CONFIG: Record<string, ChartConfig> = {
  'chain/account/commits.50-percentile': {
    type: 'line',
    title: 'Account Commits (50th Percentile)',
    description: 'Shows the median time taken for account commits',
    unit: 'ns' // Added unit
  },
  'chain/account/commits.95-percentile': {
    type: 'line',
    title: 'Account Commits (95th Percentile)',
    description: 'Shows the 95th percentile time taken for account commits',
    unit: 'ns' // Added unit
  }
  // Add units to other metrics as needed
};