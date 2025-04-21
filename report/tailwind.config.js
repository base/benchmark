import defaultTheme from "tailwindcss/defaultTheme";

/**
 * @type {import('tailwindcss').Config}
 */
export default {
  theme: {
    fontFamily: {
      sans: ["Coinbase Sans", ...defaultTheme.fontFamily.sans],
    },
    extend: {
      fontFamily: {
        // Creating specific utility classes for each font family
        "coinbase-text": ["Coinbase Text", ...defaultTheme.fontFamily.sans],
        "coinbase-display": [
          "Coinbase Display",
          ...defaultTheme.fontFamily.sans,
        ],
      },
    },
  },
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
};
