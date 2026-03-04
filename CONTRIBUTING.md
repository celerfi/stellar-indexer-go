# Contributing to CelerFi Stellar Indexer

We welcome contributions to the CelerFi Stellar Indexer. Please follow these guidelines to ensure a consistent and high-quality codebase.

## Development Workflow

1.  **Branching Strategy**: 
    *   `main`: Production-ready code.
    *   `dev`: Staging and active integration.
    *   Feature branches should be created from `dev` and named using the convention `feature/your-feature-name` or `fix/your-fix-name`.

2.  **Coding Standards**:
    *   Follow standard Go idioms and formatting (`go fmt`).
    *   Ensure all new functionality includes appropriate database schema updates.
    *   **Strict Policy**: Do not include comments within the source code unless absolutely necessary for complex algorithmic explanations. Documentation should be handled via clear naming conventions and external documentation.

3.  **Dependency Management**:
    *   Always run `go mod tidy` before submitting changes.
    *   Update `package.json` if new infrastructure management scripts are added.

## Submission Process

1.  **Pull Requests**:
    *   Target the `dev` branch for all feature contributions.
    *   Include a concise title and a detailed description of the changes.
    *   Reference the issue number being resolved.

2.  **Verification**:
    *   Ensure the project builds successfully using `go build .` or `pnpm build`.
    *   Verify that the database initialization scripts (`pnpm db:init`) function correctly with the new changes.
    *   Test GraphQL query resolution via the local playground.

## Reporting Issues

*   Provide a clear and concise title.
*   Describe the current behavior and the expected behavior.
*   Include relevant logs or error messages from the indexer or database.
