import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default {
  plugins: {
    '@tailwindcss/postcss': {
      // Explicitly specify content paths for better watch detection
      content: [
        path.join(__dirname, './src/**/*.{js,ts,jsx,tsx}'),
        path.join(__dirname, './client.tsx'),
        path.join(__dirname, '../../packages/**/*.{js,ts,jsx,tsx}'),
      ],
    },
  },
};
