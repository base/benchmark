// API configuration for the benchmark report frontend

export interface ApiConfig {
  baseUrl: string;
  endpoints: {
    metadata: string;
    metrics: (runId: string, outputDir: string, nodeType: string) => string;
    health: string;
  };
}

// Get API base URL from environment or default to localhost
const getApiBaseUrl = (): string => {
  // Check for environment variable (useful for build-time configuration)
  if (typeof window !== 'undefined') {
    // Client-side: check for runtime configuration
    const runtimeConfig = (window as any).__RUNTIME_CONFIG__;
    if (runtimeConfig?.API_BASE_URL) {
      return runtimeConfig.API_BASE_URL;
    }
  }

  // Fallback to environment variable or default
  return import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
};

export const apiConfig: ApiConfig = {
  baseUrl: getApiBaseUrl(),
  endpoints: {
    metadata: '/api/v1/metadata',
    metrics: (runId: string, outputDir: string, nodeType: string) => 
      `/api/v1/metrics/${encodeURIComponent(runId)}/${encodeURIComponent(outputDir)}/${encodeURIComponent(nodeType)}`,
    health: '/api/v1/health',
  },
};

// Helper function to build full URLs
export const buildApiUrl = (endpoint: string): string => {
  const baseUrl = apiConfig.baseUrl.replace(/\/$/, ''); // Remove trailing slash
  const cleanEndpoint = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
  return `${baseUrl}${cleanEndpoint}`;
};

// Type-safe API URL builders
export const apiUrls = {
  metadata: () => buildApiUrl(apiConfig.endpoints.metadata),
  metrics: (runId: string, outputDir: string, nodeType: string) => 
    buildApiUrl(apiConfig.endpoints.metrics(runId, outputDir, nodeType)),
  health: () => buildApiUrl(apiConfig.endpoints.health),
} as const; 