import nextJest from 'next/jest.js';

const createJestConfig = nextJest({
    // Provide the path to your Next.js app to load next.config.js and .env files
    dir: './',
});

// Add any custom config to be passed to Jest
const customJestConfig = {
    setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts'],
    testEnvironment: 'jsdom',
    moduleNameMapper: {
        '^@/(.*)$': '<rootDir>/src/$1',
    },
    collectCoverageFrom: [
        'src/**/*.{js,jsx,ts,tsx}',
        '!src/**/*.d.ts',
        '!src/app/layout.tsx',
        '!src/app/globals.css',
    ],
    testMatch: [
        '<rootDir>/src/**/__tests__/**/*.{js,jsx,ts,tsx}',
        '<rootDir>/src/**/*.{test,spec}.{js,jsx,ts,tsx}',
    ],
    transformIgnorePatterns: [
        'node_modules/(?!(react-markdown|remark|unified|bail|is-plain-obj|trough|vfile|vfile-message|mdast-util-to-hast|mdast-util-to-string|unist-util-is|unist-util-visit|unist-util-visit-parents|micromark|decode-named-character-reference|character-entities|property-information|hast-util-whitespace|space-separated-tokens|comma-separated-tokens|hast-util-is-element|hast-util-has-property|web-namespaces|zwitch|html-void-elements)/)'
    ],
};

// Export ESM default config
export default createJestConfig(customJestConfig);
