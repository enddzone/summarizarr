import { FlatCompat } from '@eslint/eslintrc';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const compat = new FlatCompat({
  baseDirectory: __dirname,
});

// Define config as a named constant to satisfy import/no-anonymous-default-export
const config = [
  {
    ignores: [
      "node_modules/**",
      ".next/**",
      "out/**",
      "build/**",
      "coverage/**",
      "next-env.d.ts",
      // Ignore legacy CommonJS Jest config if present
      "jest.config.js",
    ],
  },
  // Allow CommonJS require in Jest config file
  {
    files: ["**/jest.config.js"],
    rules: {
      "@typescript-eslint/no-require-imports": "off",
    },
  },
  ...compat.extends('next/core-web-vitals', 'next/typescript'),
];

export default config;