export default {
  plugins: {
    '@tailwindcss/postcss': {
      // Explicitly specify content paths for better watch detection
      content: [
        './src/**/*.{js,ts,jsx,tsx}',
        './client.tsx',
        './server.tsx',
        '../../../packages/**/*.{js,ts,jsx,tsx}',
      ],
    },
  },
};
