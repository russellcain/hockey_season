# Draft Room — Fantasy Hockey Draft UI

A browser-based draft room for the fantasy hockey league. Built with React 19, TypeScript, Vite, and Tailwind CSS v4.

## Prerequisites

- [Node.js 20+](https://nodejs.org/)

## Running the app

```bash
cd frontend
npm install
npm run dev
```

Opens at [http://localhost:5173](http://localhost:5173).

The draft room loads with mock data so all features are immediately interactive — no backend required.

## Running the tests

```bash
# Watch mode (re-runs on file change)
npm run test

# Single run (CI-friendly)
npm run test:run
```

Tests are written with [Vitest](https://vitest.dev/) and [React Testing Library](https://testing-library.com/). There are four test suites:

| Suite | What it covers |
|---|---|
| `filterLogic.test.ts` | Pure filter logic — position, team, hide-taken, draftable-only, search, and combinations |
| `DraftViolationModal.test.tsx` | Over-cap modal (title, cap table values, dismiss) and position-full modal (slot count, label) |
| `DraftSnackbar.test.tsx` | Draft confirmation snackbar — display, auto-dismiss at 3 s, timer cleanup on unmount |
| `AvailablePlayerList.test.tsx` | Integration tests covering filters, violation flows, and a successful draft + snackbar cycle |
