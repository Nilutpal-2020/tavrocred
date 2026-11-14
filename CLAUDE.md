# CLAUDE.md — Tavrocred, P2P Loan Tracking Platform

This file defines the engineering rules and product context for Claude Code working in this repository. Follow it before making changes.

## 1. Product context

Build a **personal P2P loan tracking application**. It helps an individual record money they lend to people they know, track repayments made outside the application, and send consent-based reminders.

This is a recordkeeping and communication product, **not a payment processor, lender, credit marketplace, collection agency, or escrow service**.

### Product decisions

- Audience: individuals managing their own personal lending.
- Frontend: web application / responsive PWA.
- Backend: Go.
- Database: PostgreSQL.
- Borrower experience: secure link; no account required for the MVP.
- Payment tracking: manual. Payments happen externally through UPI, bank transfer, cash, or other methods.
- Loan types: interest-free and interest-bearing.
- Initial interest support: no interest and simple interest. Do not implement compound interest until its conventions and repayment allocation rules are explicitly specified.
- Currency: INR by default, while keeping currency explicit on each loan.
- Architecture: modular monolith, with a separately runnable background worker.
- Do not introduce microservices, Kafka, Redis, or payment integrations without a concrete requirement and explicit approval.

## 2. Core product principles

1. **The ledger is the source of truth.** Dashboard figures, reminders, borrower views, and reports must derive from the same backend financial logic.
2. **External payments are not verified automatically.** A manually entered or borrower-reported payment is pending until the lender confirms receipt.
3. **Never imply that the platform moved or received money.** Clearly distinguish recorded, reported, and lender-confirmed payments.
4. **Borrower access is restricted.** A borrower link grants access only to the specific loan and permitted actions.
5. **Preserve history.** Do not silently overwrite agreed loan terms, confirmed repayments, acknowledgements, or audit events.
6. **No coercive recovery.** Reminders must be respectful, consent-based, rate-limited, and clearly identify the platform and the lender who initiated them.
7. **Privacy by default.** Collect only necessary data and never expose a lender's other borrowers or portfolio to a borrower.
8. **No legal overclaims.** Acknowledgement is a record of the borrower's response, not automatically a legally enforceable electronic signature or proof of money transfer.

## 3. Technology baseline

Use these choices unless the repository already has a compatible, established implementation:

### Backend

- Go (use the version pinned in `go.mod` / Dockerfile; do not upgrade casually).
- HTTP router: `github.com/go-chi/chi/v5`.
- PostgreSQL driver: `github.com/jackc/pgx/v5`.
- SQL generation: `sqlc`.
- Migrations: Goose.
- Decimal arithmetic: `github.com/shopspring/decimal`.
- Validation: `github.com/go-playground/validator/v10` where useful.
- JWT: `github.com/golang-jwt/jwt/v5`, if JWT is used.
- Password hashing: Argon2id using `golang.org/x/crypto/argon2`.
- Configuration: environment-based configuration; validate required values at startup.
- Logging: standard library `log/slog`.
- Tests: standard Go testing plus `testify` where it improves readability.

### Frontend

- Next.js with TypeScript.
- Use the package manager and lockfile already present. Do not replace them.
- Use typed API contracts generated from or kept in sync with the OpenAPI specification.
- Keep borrower-facing pages responsive and minimal.

### Infrastructure

- PostgreSQL is the primary datastore.
- Docker Compose is the local development environment.
- Use Mailpit locally for email inspection.
- Email provider integration must be behind an interface so providers can be swapped.
- Store uploaded documents in an object-storage abstraction; do not store large files directly in PostgreSQL.

## 4. Repository structure

Prefer this structure. Adapt to the repository if it already contains an established equivalent; avoid needless file moves.

```text
apps/
  web/
    app/
    components/
    hooks/
    lib/
    types/
services/
  api/
    cmd/
      api/
      worker/
    internal/
      config/
      database/
      server/
      middleware/
      modules/
        auth/
        user/
        peer/
        loan/
        repayment/
        ledger/
        acknowledgement/
        notification/
        analytics/
        audit/
      jobs/
      email/
      storage/
    migrations/
    queries/
    gen/
    go.mod
    sqlc.yaml
packages/
  contracts/
    openapi.yaml
deploy/
docs/
scripts/
docker-compose.yml
.env.example
Makefile
CLAUDE.md
```

### Backend module responsibilities

Use clear boundaries:

- `handler`: HTTP parsing, authentication context, status codes, response mapping.
- `service`: business rules and orchestration.
- `repository`: database operations.
- `model` / domain types: core concepts and invariants.
- `dto`: API request and response types.
- `validator`: input validation where not covered by domain constructors.
- `errors`: stable domain/application errors.

Handlers must not contain SQL or financial calculations. Repositories must not decide business policy.

## 5. Domain model and terminology

Use consistent terminology:

- **User / lender**: the authenticated owner of the personal lending records.
- **Peer / borrower**: a contact associated with one lender.
- **Loan**: a record of agreed lending terms.
- **Repayment**: a payment entry associated with a loan.
- **Borrower-reported payment**: a borrower says they paid; it is not yet confirmed.
- **Confirmed repayment**: lender has verified receipt of the external payment.
- **Acknowledgement**: borrower response to the loan details shown to them.
- **Dispute**: borrower says one or more displayed loan details are incorrect.
- **Notification**: an outbound communication attempt and its delivery state.

A peer belongs to one user in the MVP. Do not assume that the same peer record is globally shared across different lenders.

### Loan statuses

Use explicit, validated state transitions. Initial statuses:

- `DRAFT`
- `ACTIVE`
- `COMPLETED`
- `CANCELLED`
- `DISPUTED`

Do not allow arbitrary status changes through a generic PATCH endpoint. Implement explicit commands / service methods for transitions.

### Repayment statuses

- `PENDING_VERIFICATION`
- `CONFIRMED`
- `REJECTED`
- `REVERSED`

A repayment affects the confirmed balance only when its status is `CONFIRMED`. Reversals must preserve the original record and be auditable.

### Acknowledgement statuses

- `PENDING`
- `ACKNOWLEDGED`
- `DISPUTED`

Treat each response or agreement revision as a historical record. Do not overwrite an earlier acknowledgement.

## 6. Financial correctness requirements

These rules are mandatory:

1. Never use `float32` or `float64` for money or interest calculations.
2. Use PostgreSQL `NUMERIC` and decimal arithmetic in Go.
3. Serialize monetary values in JSON as decimal strings, for example `"1250.00"`.
4. Define rounding explicitly and consistently. Use two decimal places for INR amounts unless a documented product rule says otherwise.
5. Keep currency on each loan. Never aggregate different currencies without an explicit conversion policy.
6. Only confirmed repayments reduce the outstanding balance.
7. Do not infer payment receipt from a UPI reference, screenshot, email, or borrower report.
8. Do not silently edit an activated loan's agreed terms. Use a loan amendment workflow with a new version and, where appropriate, renewed borrower acknowledgement.
9. Define interest conventions explicitly: annual rate, accrual start/end dates, day-count basis, rounding, and repayment allocation.
10. Initial supported interest modes are:
    - `NONE`
    - `SIMPLE`
11. Compound interest, penalties, late fees, and complex amortization schedules are out of MVP scope until explicitly designed.
12. Prevent duplicate mutation effects with database constraints, transactions, and idempotency keys where appropriate.
13. Handle concurrent repayment confirmation safely. Recalculate or lock the relevant loan/ledger state inside a database transaction.
14. Define overpayment behavior before accepting a repayment that exceeds the outstanding amount. Do not silently clamp or discard the excess.

For a simple-interest loan, a possible initial convention is:

`interest = principal × annual_rate × elapsed_days / 365`

This is only valid if the loan terms explicitly select ACTUAL/365 and the product defines whether interest is calculated on original principal or reducing principal. Do not assume this convention for every loan.

### Ledger and audit history

Record meaningful events such as:

- `LOAN_CREATED`
- `LOAN_ACTIVATED`
- `LOAN_AMENDED`
- `BORROWER_ACKNOWLEDGED`
- `LOAN_DISPUTED`
- `REPAYMENT_RECORDED`
- `REPAYMENT_REPORTED`
- `REPAYMENT_CONFIRMED`
- `REPAYMENT_REJECTED`
- `REPAYMENT_REVERSED`
- `LOAN_COMPLETED`
- `LOAN_CANCELLED`

Use append-only event/audit records for important financial and agreement changes. Corrections should be represented as new events or reversal records, not destructive edits.

## 7. Database rules

- Use UUID primary keys.
- Use `TIMESTAMPTZ` for event timestamps and `DATE` for contractual calendar dates.
- Use `NUMERIC`, never floating-point, for monetary columns.
- Add foreign keys, check constraints, and indexes for common query paths.
- Enforce lender ownership in the data access layer and preferably in query predicates.
- Use migrations for every schema change. Never rely on runtime auto-migration.
- Name migrations sequentially and make them safe to run in deployment.
- Use transactions for multi-table business operations.
- Keep schema constraints aligned with domain validation.
- Do not store plaintext borrower access tokens, refresh tokens, or password credentials.
- Store only hashes of opaque access/refresh tokens.
- Do not put secrets in source control or migrations.

Before changing schema, inspect existing migrations and generated sqlc code. Update queries and regenerate code using the documented project command.

## 8. API conventions

- Prefix routes with `/api/v1`.
- Use JSON request and response bodies.
- Keep response and error envelopes consistent.
- Return stable machine-readable error codes.
- Validate all inputs server-side.
- Use pagination for collection endpoints.
- Use ISO 8601 timestamps and `YYYY-MM-DD` for date-only fields.
- Represent money as decimal strings.
- Use `Idempotency-Key` for operations where client retries could create duplicate financial records.
- Do not trust IDs supplied by the frontend: every resource query must enforce ownership.
- Public borrower endpoints must authorize using a restricted token/session, never by a loan UUID alone.
- Do not expose internal database errors or secrets in API responses.
- Keep `packages/contracts/openapi.yaml` in sync with implemented routes.

Suggested route groups:

```text
/api/v1/auth/*
/api/v1/dashboard/*
/api/v1/peers/*
/api/v1/loans/*
/api/v1/loans/{loanID}/repayments/*
/api/v1/repayments/*
/api/v1/notifications/*
/api/v1/reports/*

/api/v1/public/borrower/access
/api/v1/public/borrower/loan
/api/v1/public/borrower/acknowledge
/api/v1/public/borrower/dispute
/api/v1/public/borrower/repayment-reported
```

## 9. Borrower secure-link security

Borrowers do not need an account for the MVP, so token security is critical.

- Generate tokens using `crypto/rand` with at least 32 random bytes.
- Store only a cryptographic hash of the token.
- Scope every token to a specific peer, loan, purpose, and expiry.
- Use separate purposes/scopes for viewing, acknowledging, disputing, and reporting repayment.
- Use short-lived, single-use tokens for sensitive actions.
- Consider exchanging an email link token for a short-lived restricted borrower session.
- Do not put loan amounts or personal information in URLs.
- Avoid consuming single-use tokens on HTTP GET because email security scanners may open links. Require an explicit user action for token exchange or acknowledgement.
- Rate-limit token exchange and borrower actions.
- Revoke tokens after use, expiry, or relevant loan state changes.
- Avoid logging raw tokens, complete tokenized URLs, or sensitive borrower details.
- Record the acknowledgement content/version and timestamp.
- Treat IP address and user-agent as optional security metadata; do not present them as identity proof.

Borrower pages must show only the relevant loan and permitted actions. They must never expose lender dashboard data, other peers, or other loans.

## 10. Notifications and background jobs

- Never send email synchronously inside a loan or repayment HTTP request.
- Use a PostgreSQL-backed transactional outbox/job table for the initial implementation.
- Insert the business change and notification job in the same database transaction.
- Use `FOR UPDATE SKIP LOCKED` or an equivalent safe claim mechanism for concurrent workers.
- Implement retry with exponential backoff and a maximum attempt count.
- Recover jobs left in `PROCESSING` after worker crashes.
- Track provider message IDs and delivery state where available.
- Make job handlers idempotent.
- Do not enqueue duplicate reminders for the same loan, recipient, reminder type, and scheduled occurrence.
- Stop or suppress reminders for completed, cancelled, or disputed loans as appropriate.
- Respect borrower notification consent and lender reminder preferences.
- Provide a reasonable reminder frequency cap and a way to pause reminders.
- Emails must clearly identify the platform and the lender who initiated the message. Never impersonate a lender or use threatening, shaming, or coercive language.

Use Mailpit for local development. Keep the email provider behind an interface.

## 11. Authentication and application security

- Hash passwords with Argon2id; never store plaintext passwords.
- Use short-lived access tokens and rotating refresh tokens.
- Store refresh-token hashes and support revocation.
- Prefer secure, HttpOnly, SameSite cookies for browser sessions where compatible with the frontend architecture. Avoid exposing long-lived tokens to JavaScript.
- Validate CORS origins explicitly.
- Apply request body size limits and timeouts.
- Rate-limit login, registration, password reset, and public borrower endpoints.
- Use parameterized SQL only.
- Do not log passwords, tokens, secrets, or full sensitive financial payloads.
- Use TLS in deployed environments.
- Keep database and mail administration ports private in production.
- Validate uploaded file type, size, and ownership if file uploads are added.
- Add security headers and CSRF protections where cookie-based authentication requires them.

## 12. Frontend conventions

- Use TypeScript strict mode.
- Keep API calls in a shared typed client layer.
- Do not duplicate business calculations in the browser. Display server-calculated balances and breakdowns.
- Use accessible forms, labels, keyboard navigation, and clear validation messages.
- Show payment verification state prominently.
- Clearly distinguish pending, confirmed, rejected, and reversed payments.
- Do not present a borrower-reported payment as money received.
- Use responsive layouts for desktop and mobile.
- Keep borrower pages minimal and mobile-friendly.
- Never put secrets in `NEXT_PUBLIC_*` environment variables.
- Do not store sensitive borrower tokens in persistent browser storage.

## 13. Testing requirements

Every meaningful feature must include tests.

### Financial tests

- Interest-free loan balance.
- Simple interest with the configured day-count convention.
- Date boundaries and leap years.
- Decimal rounding.
- Partial repayment.
- Multiple repayments.
- Pending repayment does not reduce balance.
- Rejected repayment does not reduce balance.
- Reversed repayment restores the balance correctly.
- Overpayment behavior.
- Repayment allocation between interest and principal.
- Concurrent confirmations do not double-count.

### Authorization and security tests

- User A cannot read or mutate User B's peers or loans.
- Borrower token cannot access a different loan.
- Expired, revoked, used, or wrong-purpose token is rejected.
- Borrower cannot access lender-only routes.
- Token is not consumed by an email scanner-style GET request.
- Rate limits apply to public token endpoints.

### Notification tests

- Completed/disputed loans do not receive inappropriate reminders.
- Duplicate reminder jobs are prevented.
- Failed provider calls are retried.
- Stale processing jobs are recovered.
- Worker retries do not send duplicate messages where provider idempotency is available.

Use unit tests for domain logic and integration tests against PostgreSQL for repository and transaction behavior. Do not mock away the financial invariants.

## 14. Development workflow

Before implementation:

1. Inspect the repository and existing conventions.
2. Read this file and relevant module documentation.
3. Identify the smallest vertical slice that delivers value.
4. State the files and migrations expected to change.
5. Implement the change with tests.
6. Run formatting, linting, tests, and relevant generation commands.
7. Summarize what changed, commands run, test results, and any remaining risks.

Preferred commands, if configured:

```bash
gofmt -w .
go test ./...
go vet ./...
sqlc generate
goose -dir migrations postgres "$DATABASE_URL" up
docker compose config
docker compose up --build
```

For frontend changes, run the repository's configured lint, typecheck, and test scripts.

Do not claim tests passed unless they were actually run and passed.

### Change discipline

- Make focused changes; avoid unrelated refactors.
- Do not add dependencies without explaining why they are needed.
- Do not change public API contracts without updating OpenAPI and frontend types.
- Do not edit generated code manually.
- Do not delete migrations that may have been applied.
- Do not change financial semantics without tests and documentation.
- Do not add payment gateway, UPI collection, lending marketplace, credit scoring, or debt-collection features to the MVP.
- Ask before making irreversible changes or introducing a materially different product/regulatory scope.

## 15. Definition of done

A feature is done only when:

- Its behavior matches the product requirements.
- Authorization and ownership checks are implemented.
- Validation and error handling are present.
- Database changes have migrations and constraints.
- Financial changes are transactional and auditable.
- Relevant unit/integration tests pass.
- OpenAPI and frontend types are updated when applicable.
- Logs do not leak sensitive data.
- Documentation and configuration examples are updated.
- The final response states what was implemented and what remains.

## 16. First implementation task

Start with **Milestone 1: repository foundation and a vertical slice for authentication plus borrower management**.

Deliver:

1. Repository skeleton matching the agreed structure.
2. Go API and worker entrypoints.
3. Configuration loading and validation.
4. PostgreSQL connection pool and health/readiness endpoints.
5. Goose migration setup.
6. Initial `users` and `peers` schema with ownership constraints.
7. Chi router, request ID, structured logging, panic recovery, and timeout middleware.
8. Registration, login, refresh, and logout APIs with secure password/token handling.
9. Peer create/list/get/update APIs with strict owner scoping.
10. Docker Compose for PostgreSQL, API, worker, web, and Mailpit.
11. `.env.example`, Makefile, README, and initial OpenAPI contract.
12. Unit and PostgreSQL integration tests.

Do not implement the full loan engine, reminders, or borrower portal in this first task. Finish the foundation cleanly, run the tests, and report the exact commands and results before proceeding to the next milestone.
