import { useCallback, useState } from "react";
import { useSearchParams } from "react-router-dom";

/**
 * Hook for storing state in URL search parameters as base64 encoded JSON
 * Supports multiple instances without conflicts
 * @param paramName The name of the search parameter
 * @param defaultValue The default value to use if the parameter is not present
 * @returns A tuple containing the state value and a setter function
 */
export function useSearchParamsState<T>(
  paramName: string,
  defaultValue: T,
): [T, (value: T | ((prevValue: T) => T)) => void] {
  const [searchParams, setSearchParams] = useSearchParams();

  // Initialize from URL or default
  const [state, setState] = useState<T>(() => {
    const paramValue = searchParams.get(paramName);
    if (paramValue) {
      try {
        // Decode base64 back to JSON string and then parse
        const jsonString = atob(paramValue);
        return JSON.parse(jsonString) as T;
      } catch (e) {
        console.error(
          `Error parsing state from URL parameter ${paramName}:`,
          e,
        );
        return defaultValue;
      }
    }

    return defaultValue;
  });

  // Custom setter that updates both state and URL
  const setStateWithSearchParams = useCallback(
    (value: T | ((prevValue: T) => T)) => {
      setState((prev) => {
        const newValue =
          typeof value === "function" ? (value as (prev: T) => T)(prev) : value;

        // Update URL immediately, keeping all other parameters
        setSearchParams(
          (currentParams) => {
            const newParams = new URLSearchParams(currentParams);
            newParams.set(paramName, btoa(JSON.stringify(newValue)));
            return newParams;
          },
          { replace: true },
        );

        return newValue;
      });
    },
    [paramName, setSearchParams],
  );

  return [state, setStateWithSearchParams];
}
