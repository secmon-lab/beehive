// Minimal ESLint flat config. We rely on TypeScript's own type-check
// for correctness and only use ESLint as a style guard; expanding the
// ruleset is a later concern.
export default [
  {
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: "module",
    },
    rules: {
      "no-debugger": "error",
    },
  },
];
