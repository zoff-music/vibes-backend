/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './src/**/*.{js,ts,jsx,tsx}',
    './client.tsx',
    '../../../packages/**/*.{js,ts,jsx,tsx}',
  ],
  // Tailwind CSS v4 uses CSS-first configuration
  // Most config is in src/index.css via @theme directive
};
