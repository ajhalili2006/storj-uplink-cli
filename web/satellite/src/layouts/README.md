# Layouts

This folder defines the three top-level app layouts.

## Layouts
- auth/AuthLayout.vue
  - Unauthenticated routes (login, signup, password recovery, activation).
  - Uses AuthBar and the shared app view container.
- account/AccountLayout.vue
  - Authenticated, account-level routes with no specific project selected.
  - Uses AppShell with AccountNav.
- project/ProjectLayout.vue
  - Authenticated, project-scoped routes.
  - Uses AppShell with ProjectNav and handles project selection/worker setup.

## Routing
Route definitions live in src/router/index.ts and map to one of the three layouts above.
