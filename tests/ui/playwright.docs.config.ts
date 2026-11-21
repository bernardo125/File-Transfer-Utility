import base from './playwright.config';

// Generates the README screenshots from the real app: `npm run docs:media`.
export default { ...base, testDir: './docs', projects: [base.projects![0]], reporter: 'list' as const, retries: 0 };
