import base from './playwright.config';

// Generates the README demo video from the real app: `npm run docs:video`.
export default { ...base, testDir: './docs', projects: [base.projects![0]], reporter: 'list' as const, retries: 0 };
