/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./src/**/*.{js,ts,jsx,tsx}",
    "./client.tsx",
    "./server.tsx",
    "../../../packages/**/*.{js,ts,jsx,tsx}",
  ],
  // Enable all variants to ensure responsive classes are generated
  safelist: [
    // Ensure responsive variants are always included
    'hidden',
    'inline',
    'sm:hidden',
    'sm:inline',
    'md:hidden',
    'md:inline',
    'lg:hidden',
    'lg:inline',
    'xl:hidden',
    'xl:inline',
  ],
  // Tailwind CSS v4 uses CSS-first configuration
  // Most config is in src/index.css via @theme directive
};