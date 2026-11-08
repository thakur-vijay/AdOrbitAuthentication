# AdOrbit Authentication Microservice — Complete Testing & Quality Assurance Specification

**Document Type:** Strict AI Agent Execution Specification  
**Project:** AdOrbit Authentication Microservice  
**Language:** Go  
**Testing Standard:** Production-Grade  
**Primary Objective:** Comprehensive Test Coverage, Reliability, Security, and Automated CI/CD

---

# 1. PRIMARY MISSION

You are acting as a **Senior Go Backend Engineer, Test Architect, Security Engineer, and QA Automation Engineer**.

Your responsibility is to thoroughly inspect, understand, test, validate, and review the entire AdOrbit Authentication Microservice.

Your task is NOT limited to writing basic unit tests.

You must establish a complete testing strategy covering every implemented component, including:

1. Unit Testing
2. Service Layer Testing
3. Repository Testing
4. PostgreSQL Integration Testing
5. RabbitMQ Integration Testing
6. HTTP Handler Testing
7. Middleware Testing
8. JWT and Session Testing
9. OTP Testing
10. Security Testing
11. End-to-End Testing
12. Concurrency and Race Condition Testing
13. Application Configuration Testing
14. Application Lifecycle Testing
15. Performance and Benchmark Testing
16. Error Handling and Failure Recovery Testing
17. Regression Testing

After implementing all tests, you MUST perform repeated reviews to identify missing test cases, weak assertions, untested branches, uncovered edge cases, and architectural testing gaps.

Continue reviewing and improving the test suite until all applicable testing requirements are satisfied.

Finally, create and configure an automated GitHub Actions workflow inside:

`.github/workflows/`

The workflow must automatically execute the appropriate testing and quality checks.

**Do not consider the task complete merely because test files have been created. The tests must actually execute and pass.**

---

# 2. STRICT AGENT RULES

These instructions are mandatory.

## 2.1 Understand the Existing Architecture First

Before writing any test:

- Inspect the complete repository structure.
- Read `go.mod` and identify the actual Go version and dependencies.
- Inspect every production package.
- Understand the application entry point.
- Understand dependency injection and initialization.
- Understand the service, repository, handler, middleware, and messaging layers.
- Inspect database schemas and migrations.
- Inspect PostgreSQL queries and transaction boundaries.
- Inspect RabbitMQ exchanges, queues, routing keys, message formats, and consumer behavior.
- Inspect JWT generation, validation, rotation, and session persistence.
- Inspect OTP generation, storage, expiration, and verification.
- Inspect configuration and environment variable handling.
- Inspect error types and HTTP error mapping.
- Inspect existing tests, if any.

Build an understanding of how requests flow through the application before implementing tests.

**Never assume that the architecture described in this document perfectly matches the current source code. The actual implementation is the source of truth.**

If a feature listed in this document does not exist, do not invent its implementation merely to satisfy a checklist.

Mark it as:

`N/A — Not implemented in the current architecture.`

If the implementation contradicts an expected behavior, report the discrepancy instead of silently changing the expected behavior.

## 2.2 Do Not Modify Production Logic Without Justification

Your primary responsibility is testing.

You must not:

- Rewrite working production code unnecessarily.
- Change business logic merely to make tests pass.
- Modify API contracts without justification.
- Remove existing functionality.
- Weaken security validations.
- Disable error handling to simplify testing.
- Introduce unnecessary dependencies.
- Change database schemas just to improve coverage.
- Introduce artificial interfaces solely to make trivial tests easier.

If testing exposes an actual production bug:

1. Identify the root cause.
2. Explain the incorrect behavior.
3. Determine the expected behavior from the existing architecture.
4. Implement the smallest justified correction.
5. Add a regression test.
6. Execute the relevant test suite again.

Do not hide production defects by weakening assertions.

## 2.3 No Fake or Meaningless Tests

Every test must verify actual behavior.

Strictly prohibited:

- Tests containing no meaningful assertions.
- Tests that only check `err == nil` when the returned result also needs validation.
- Tests that merely execute a function without checking its outcome.
- Tests that mock the functionality they are supposed to verify.
- Tests that always return predefined successful responses without validating interactions.
- Tests that ignore returned errors.
- Tests that skip difficult failure scenarios without documenting why.
- Tests written exclusively to increase coverage percentages.

Every test must have:

- A clear purpose.
- Meaningful input.
- Expected output.
- Expected error behavior.
- Relevant state verification.
- Appropriate cleanup.

## 2.4 Follow Go Testing Best Practices

Use:

- Standard Go `testing` package.
- Table-driven tests wherever appropriate.
- `t.Run()` for individual scenarios.
- `t.Helper()` inside reusable test helpers.
- `t.Cleanup()` for resource cleanup.
- `context.Context` for cancellation and timeout testing.
- `httptest` for HTTP testing.
- Test fixtures and factories for reusable data.
- Explicit error assertions.
- Deterministic test data.

Use `t.Parallel()` only when tests are genuinely isolated and safe to execute concurrently.

Do not introduce unnecessary abstractions into test code.

Keep tests readable, maintainable, and easy to debug.

---

# 3. TESTING ARCHITECTURE AND DEPENDENCY RULES

## 3.1 Unit Testing

Unit tests must isolate the component being tested.

Mock external dependencies such as:

- Database repositories.
- RabbitMQ publishers and consumers.
- External service clients.
- Email delivery providers.
- Clock/time dependencies where required.
- Other independent services.

Use the project's existing mocking strategy where possible.

If an appropriate mocking library is already installed, reuse it.

Do not introduce multiple mocking frameworks without a strong reason.

## 3.2 Integration Testing

Integration tests must validate actual interactions between components.

Use real infrastructure wherever practical.

Required infrastructure:

- PostgreSQL.
- RabbitMQ.

Preferred approach:

`testcontainers-go`

If the repository already uses Docker Compose for test infrastructure, evaluate whether it can be reused.

Integration tests must:

- Create isolated test resources.
- Avoid relying on developer databases.
- Avoid depending on production credentials.
- Apply the actual database migrations.
- Clean up test data.
- Use deterministic fixtures.
- Fail clearly when required infrastructure is unavailable.
- Never silently pass because infrastructure could not start.

## 3.3 End-to-End Testing

E2E tests must validate complete workflows across the actual application boundaries.

Do not replace all internal components with mocks and call that an E2E test.

Where complete external infrastructure is impractical, clearly distinguish:

- Component integration tests.
- Workflow integration tests.
- True end-to-end tests.

## 3.4 Test Isolation

Every test must be independent.

Never assume:

- A previous test has already inserted data.
- Tests execute in a particular order.
- Database records already exist.
- RabbitMQ queues contain expected messages.
- A previous test has initialized global state.

Use unique test identifiers where necessary.

Ensure cleanup executes even when assertions fail.

---

# 4. UNIT TESTING

## 4.1 Authentication Service Tests

Test every implemented public service method.

### Signup()

- [ ] Successful signup.
- [ ] Email already registered.
- [ ] Existing pending signup with the same password.
- [ ] Existing pending signup with a different password.
- [ ] Existing failed signup.
- [ ] Password hashing failure.
- [ ] Credential saving failure.
- [ ] User creation failure.
- [ ] Unexpected user ID returned by Users Service.
- [ ] Credential status update failure.
- [ ] OTP sending failure.
- [ ] Already verified email handling.
- [ ] Context cancellation.
- [ ] Database timeout.
- [ ] RabbitMQ timeout.
- [ ] Duplicate signup requests.
- [ ] Partial signup failure and recovery.

### Login()

- [ ] Successful login.
- [ ] Invalid email.
- [ ] Invalid password.
- [ ] Active account.
- [ ] Pending account.
- [ ] Failed account.
- [ ] Verified email.
- [ ] Unverified email.
- [ ] User fetch failure.
- [ ] Session creation failure.
- [ ] OTP sending failure, if applicable.
- [ ] Invalid credential status.
- [ ] Context cancellation.
- [ ] Database timeout.
- [ ] Correct user identity and role propagation.

### VerifyOTP()

- [ ] Successful OTP verification.
- [ ] Incorrect OTP.
- [ ] Expired OTP.
- [ ] Email not found.
- [ ] Credential not found.
- [ ] OTP verification failure.
- [ ] Email verification update failure.
- [ ] User fetch failure.
- [ ] Session creation failure.
- [ ] Context cancellation.
- [ ] Repeated verification attempts.

### ResendOTP()

- [ ] Successful OTP resend.
- [ ] Email not found.
- [ ] Inactive account.
- [ ] Already verified email.
- [ ] Resend cooldown active.
- [ ] Maximum resend limit exceeded.
- [ ] OTP sending failure.
- [ ] Previous OTP invalidation.
- [ ] Context cancellation.

### Logout()

- [ ] Successful logout.
- [ ] Invalid session.
- [ ] Already revoked session.
- [ ] JWT service failure.
- [ ] Context cancellation.
- [ ] Repeated logout request.

### Refresh()

- [ ] Successful token refresh.
- [ ] Invalid refresh token.
- [ ] Expired refresh token.
- [ ] Revoked refresh token.
- [ ] Reused refresh token.
- [ ] JWT service failure.
- [ ] Session not found.
- [ ] Context cancellation.
- [ ] Concurrent refresh requests.
- [ ] Old refresh token invalidation.
- [ ] Correct session and user association.

---

# 5. JWT AND SESSION TESTING

## 5.1 Access Token

- [ ] Generate access token.
- [ ] Validate valid access token.
- [ ] Expired access token.
- [ ] Invalid signature.
- [ ] Malformed token.
- [ ] Missing claims.
- [ ] Invalid issuer.
- [ ] Invalid audience.
- [ ] Invalid user ID claim.
- [ ] Invalid token type.
- [ ] Incorrect signing algorithm.
- [ ] Invalid role claim.
- [ ] Token expiration boundary.
- [ ] Tampered token.

## 5.2 Refresh Token

- [ ] Generate refresh token.
- [ ] Validate valid refresh token.
- [ ] Expired refresh token.
- [ ] Invalid refresh token.
- [ ] Refresh token rotation.
- [ ] Old refresh token invalidation.
- [ ] Refresh token reuse detection.
- [ ] Concurrent refresh requests.
- [ ] Invalid token family/session association.
- [ ] Revoked refresh token.
- [ ] Token tampering.

## 5.3 Session Management

- [ ] Create session.
- [ ] Fetch session.
- [ ] Revoke session.
- [ ] Logout session.
- [ ] Refresh revoked session.
- [ ] Session expiration.
- [ ] Multiple sessions for one user.
- [ ] Concurrent session creation.
- [ ] Session ownership validation.
- [ ] Invalid session identifier.
- [ ] Repeated session revocation.

Verify that sensitive token material is not stored or exposed contrary to the security design.

---

# 6. OTP SERVICE TESTING

## 6.1 OTP Generation

- [ ] Generate OTP successfully.
- [ ] Verify OTP length.
- [ ] Verify OTP format.
- [ ] Verify cryptographically secure randomness.
- [ ] OTP collision handling, if applicable.
- [ ] Leading-zero OTP handling.
- [ ] Boundary values.

## 6.2 OTP Storage

- [ ] Store OTP successfully.
- [ ] Store hashed OTP.
- [ ] Database failure.
- [ ] Expiration timestamp.
- [ ] Previous OTP invalidation.
- [ ] Correct email/user association.
- [ ] Duplicate storage behavior.

## 6.3 OTP Verification

- [ ] Correct OTP.
- [ ] Incorrect OTP.
- [ ] Expired OTP.
- [ ] Already used OTP.
- [ ] Non-existent OTP.
- [ ] Maximum attempts exceeded.
- [ ] OTP verification after expiration.
- [ ] Concurrent verification requests.
- [ ] Correct one-time-use behavior.

## 6.4 OTP Resend

- [ ] Successful resend.
- [ ] Cooldown validation.
- [ ] Rate limit exceeded.
- [ ] Previous OTP invalidation.
- [ ] New OTP generation.
- [ ] Storage failure.
- [ ] Delivery failure.
- [ ] Concurrent resend requests.

---

# 7. SECURITY PACKAGE TESTING

## 7.1 Password Hashing

- [ ] Successful password hashing.
- [ ] Same password produces different hashes.
- [ ] Correct password verification.
- [ ] Incorrect password verification.
- [ ] Invalid hash handling.
- [ ] Empty password handling.
- [ ] Password hash format validation.
- [ ] Argon2id parameter validation.
- [ ] Malformed encoded hash.
- [ ] Extremely long password handling.
- [ ] No plaintext password exposure.

Verify that the implementation uses the intended Argon2id configuration.

Do not weaken production password hashing parameters for tests.

## 7.2 Input Validation

- [ ] Valid email.
- [ ] Invalid email.
- [ ] Empty email.
- [ ] Valid password.
- [ ] Invalid password.
- [ ] Weak password.
- [ ] Invalid role.
- [ ] Empty required fields.
- [ ] Oversized input.
- [ ] Malformed input.
- [ ] Whitespace-only input.
- [ ] Unicode input.
- [ ] Boundary-length input.

## 7.3 Security Regression Tests

Check for:

- [ ] Password leakage in API responses.
- [ ] Password leakage in logs.
- [ ] Refresh token leakage in logs.
- [ ] OTP leakage in logs.
- [ ] JWT secret leakage.
- [ ] Sensitive database error exposure.
- [ ] User enumeration through inconsistent responses, where relevant.
- [ ] Authentication bypass.
- [ ] Authorization bypass.
- [ ] Incorrect token acceptance.
- [ ] Replay attacks against one-time credentials.
- [ ] Missing request size limits.
- [ ] Malformed authorization headers.
- [ ] Algorithm confusion vulnerabilities.
- [ ] Timing-sensitive password comparison issues.

Do not claim that a security property is verified merely because the corresponding function was executed.

---

# 8. POSTGRESQL REPOSITORY INTEGRATION TESTING

All repository integration tests must use a real PostgreSQL instance.

Apply actual project migrations.

## 8.1 Credential Repository

### SaveCredentials()

- [ ] Insert credentials successfully.
- [ ] Duplicate email handling.
- [ ] Database unavailable.
- [ ] Invalid data.
- [ ] Unique constraint validation.
- [ ] Transaction rollback.
- [ ] Correct default values.
- [ ] Correct timestamps.

### FetchByEmail()

- [ ] Existing email.
- [ ] Non-existing email.
- [ ] Database error.
- [ ] Invalid email input.
- [ ] Case sensitivity behavior, according to schema rules.

### FetchByID()

- [ ] Existing user ID.
- [ ] Non-existing user ID.
- [ ] Database error.
- [ ] Invalid user ID.

### UpdateCredentialStatus()

- [ ] Pending to Active.
- [ ] Active to Active.
- [ ] Non-existing credential.
- [ ] Invalid status.
- [ ] Database error.
- [ ] Correct persisted state.

### UpdateIsEmailVerified()

- [ ] Successfully verify email.
- [ ] Already verified email.
- [ ] Non-existing credential.
- [ ] Database error.
- [ ] Correct persisted state.

## 8.2 Session Repository

Test all implemented session repository methods.

- [ ] Create session.
- [ ] Fetch session.
- [ ] Fetch by refresh token hash.
- [ ] Revoke session.
- [ ] Update refresh token.
- [ ] Expired session behavior.
- [ ] Invalid session ID.
- [ ] Duplicate session behavior.
- [ ] Database failure.
- [ ] Concurrent updates.
- [ ] Correct transaction behavior.

## 8.3 OTP Repository

Test all implemented OTP repository methods.

- [ ] Save OTP.
- [ ] Fetch OTP.
- [ ] Update OTP.
- [ ] Delete or invalidate OTP.
- [ ] Expiration handling.
- [ ] Used OTP handling.
- [ ] Maximum attempts handling.
- [ ] Database failure.
- [ ] Concurrent verification updates.

## 8.4 Database Integrity

- [ ] Unique email constraint.
- [ ] Primary key constraint.
- [ ] NOT NULL constraints.
- [ ] Foreign key constraints, where applicable.
- [ ] Transaction rollback.
- [ ] Concurrent signup with same email.
- [ ] Concurrent database writes.
- [ ] Migration up.
- [ ] Migration down.
- [ ] Migration consistency.
- [ ] Index behavior for important queries.

Never test database behavior using mocked SQL responses when the purpose is to verify actual SQL behavior.

---

# 9. RABBITMQ TESTING

## 9.1 Connection Management

- [ ] Successful connection.
- [ ] Invalid RabbitMQ URL.
- [ ] RabbitMQ unavailable.
- [ ] Channel creation failure.
- [ ] Connection closure.
- [ ] Channel closure.
- [ ] Context cancellation.
- [ ] Reconnection behavior, if implemented.

## 9.2 Queue Management

- [ ] Queue declaration.
- [ ] Durable queue configuration.
- [ ] Queue declaration failure.
- [ ] Missing queue handling.
- [ ] Correct exchange configuration.
- [ ] Correct routing key.
- [ ] Correct message persistence settings.

## 9.3 CreateUser RPC

- [ ] Successful user creation.
- [ ] User already exists.
- [ ] Invalid user request.
- [ ] Missing user ID.
- [ ] Missing role.
- [ ] Users Service error.
- [ ] Invalid response JSON.
- [ ] Correlation ID mismatch.
- [ ] Unexpected user ID.
- [ ] RPC timeout.
- [ ] Context cancellation.
- [ ] Duplicate message handling.
- [ ] Malformed message handling.

## 9.4 FetchUser RPC

- [ ] Successful user fetch.
- [ ] User not found.
- [ ] Invalid user ID.
- [ ] Invalid role.
- [ ] Unexpected user ID.
- [ ] Unexpected user role.
- [ ] Invalid response JSON.
- [ ] Correlation ID mismatch.
- [ ] RPC timeout.
- [ ] Context cancellation.

## 9.5 Messaging Reliability

Where applicable, verify:

- [ ] Request/reply correlation.
- [ ] Correct response matching.
- [ ] Timeout cleanup.
- [ ] Consumer acknowledgement behavior.
- [ ] Negative acknowledgement behavior.
- [ ] Duplicate message handling.
- [ ] Publisher error propagation.
- [ ] Channel failure handling.
- [ ] Resource cleanup.
- [ ] No indefinite waiting for responses.

Use real RabbitMQ for messaging integration tests.

Do not rely exclusively on mocked publishers to claim RabbitMQ integration coverage.

---

# 10. HTTP HANDLER TESTING

Use Go's standard `net/http/httptest` package.

Mock the service dependency at the handler boundary.

Test both successful and unsuccessful responses.

## 10.1 Signup Endpoint

- [ ] Successful signup.
- [ ] Invalid request body.
- [ ] Malformed JSON.
- [ ] Missing required fields.
- [ ] Invalid email.
- [ ] Invalid password.
- [ ] Invalid role.
- [ ] Email conflict.
- [ ] Internal server error.
- [ ] Correct response JSON.
- [ ] Correct HTTP status code.
- [ ] Correct response headers.
- [ ] No sensitive information in response.

## 10.2 Login Endpoint

- [ ] Successful login.
- [ ] Invalid credentials.
- [ ] Invalid request body.
- [ ] Missing required fields.
- [ ] Pending account.
- [ ] Failed account.
- [ ] Internal server error.
- [ ] Correct response JSON.
- [ ] Correct HTTP status code.
- [ ] No sensitive information in response.

## 10.3 Verify OTP Endpoint

- [ ] Successful verification.
- [ ] Invalid OTP.
- [ ] Expired OTP.
- [ ] Invalid request body.
- [ ] Missing email.
- [ ] Missing OTP.
- [ ] Internal server error.
- [ ] Correct response JSON.
- [ ] Correct HTTP status code.

## 10.4 Resend OTP Endpoint

- [ ] Successful resend.
- [ ] Invalid email.
- [ ] Account inactive.
- [ ] Already verified account.
- [ ] Cooldown active.
- [ ] Rate limit exceeded.
- [ ] Internal server error.

## 10.5 Refresh Endpoint

- [ ] Successful token refresh.
- [ ] Missing refresh token.
- [ ] Invalid refresh token.
- [ ] Expired refresh token.
- [ ] Revoked refresh token.
- [ ] Internal server error.
- [ ] Correct response structure.

## 10.6 Logout Endpoint

- [ ] Successful logout.
- [ ] Missing authorization token.
- [ ] Invalid access token.
- [ ] Expired access token.
- [ ] Already revoked session.
- [ ] Internal server error.
- [ ] Correct HTTP status code.

## 10.7 General HTTP Tests

- [ ] Invalid HTTP method.
- [ ] Invalid Content-Type.
- [ ] Empty request body.
- [ ] Malformed JSON.
- [ ] Oversized request body.
- [ ] Correct status codes.
- [ ] Correct response headers.
- [ ] Correct error response format.
- [ ] Unknown route behavior.
- [ ] Unsupported request encoding.
- [ ] Multiple JSON values in one request body, where relevant.

---

# 11. MIDDLEWARE TESTING

## 11.1 JWT Authentication Middleware

- [ ] Valid access token.
- [ ] Missing authorization header.
- [ ] Invalid authorization format.
- [ ] Invalid JWT.
- [ ] Expired JWT.
- [ ] Invalid signature.
- [ ] Revoked session.
- [ ] Missing user ID claim.
- [ ] Missing role claim.
- [ ] Correct principal/context population.
- [ ] Incorrect token type.
- [ ] Malformed bearer token.
- [ ] Correct unauthorized response.

## 11.2 Role Authorization Middleware

- [ ] Advertiser access allowed.
- [ ] Publisher access allowed.
- [ ] Admin access allowed.
- [ ] Unauthorized role.
- [ ] Missing role claim.
- [ ] Multiple allowed roles.
- [ ] Role mismatch.
- [ ] Correct forbidden response.
- [ ] Authentication and authorization distinction.

## 11.3 Rate Limiter Middleware

- [ ] Request under rate limit.
- [ ] Request limit exceeded.
- [ ] Rate limit reset.
- [ ] Concurrent requests.
- [ ] Correct HTTP 429 response.
- [ ] Independent client limits, if supported.
- [ ] Correct expiration behavior.

## 11.4 Request ID Middleware

- [ ] Generate request ID.
- [ ] Preserve existing request ID.
- [ ] Request ID in response header.
- [ ] Request ID available to downstream handlers.

## 11.5 Error Handling Middleware

- [ ] Map validation errors.
- [ ] Map unauthorized errors.
- [ ] Map forbidden errors.
- [ ] Map conflict errors.
- [ ] Map not found errors.
- [ ] Map internal errors.
- [ ] Hide internal implementation details.
- [ ] Preserve appropriate status codes.
- [ ] Handle unknown errors safely.

---

# 12. END-TO-END AND WORKFLOW INTEGRATION TESTING

Validate complete authentication workflows across the actual application boundaries.

## 12.1 Signup Workflow

- [ ] Signup request.
- [ ] Credential creation.
- [ ] User creation through RabbitMQ.
- [ ] Credential status update.
- [ ] OTP generation.
- [ ] OTP storage.
- [ ] Successful signup response.
- [ ] Correct persisted state.

## 12.2 Email Verification Workflow

- [ ] Request OTP.
- [ ] Retrieve OTP through a test delivery mechanism.
- [ ] Verify OTP.
- [ ] Update email verification status.
- [ ] Generate access token.
- [ ] Generate refresh token.
- [ ] Create session.
- [ ] Verify persisted state.

Never depend on a real external email provider for automated tests.

Use a controlled test email implementation or test delivery mechanism.

## 12.3 Login Workflow

- [ ] Login with valid credentials.
- [ ] Verify password.
- [ ] Fetch user details.
- [ ] Check email verification status.
- [ ] Create session.
- [ ] Return tokens.
- [ ] Verify correct user identity and role.

## 12.4 Refresh Workflow

- [ ] Submit refresh token.
- [ ] Validate refresh token.
- [ ] Validate session.
- [ ] Rotate refresh token.
- [ ] Invalidate old refresh token.
- [ ] Return new token pair.
- [ ] Verify old token rejection.

## 12.5 Logout Workflow

- [ ] Submit logout request.
- [ ] Validate access token.
- [ ] Revoke session.
- [ ] Verify subsequent refresh failure.
- [ ] Verify subsequent authenticated access behavior according to the implementation.

## 12.6 Failure Workflows

- [ ] User creation failure.
- [ ] Database failure.
- [ ] RabbitMQ failure.
- [ ] OTP storage failure.
- [ ] JWT generation failure.
- [ ] Invalid user state.
- [ ] Partial signup failure recovery.
- [ ] Duplicate signup recovery.
- [ ] Context timeout.
- [ ] Context cancellation.

---

# 13. CONCURRENCY AND RACE CONDITION TESTING

Concurrency testing is mandatory for critical authentication operations.

Test:

- [ ] Two simultaneous signups using the same email.
- [ ] Multiple simultaneous login attempts.
- [ ] Concurrent OTP verification.
- [ ] Concurrent OTP resend.
- [ ] Concurrent refresh requests with the same token.
- [ ] Concurrent logout and refresh.
- [ ] Concurrent session creation.
- [ ] Concurrent repository writes.
- [ ] Concurrent RabbitMQ requests.
- [ ] Concurrent access to shared in-memory state.
- [ ] Concurrent rate limiter requests.

Verify that:

- Duplicate operations do not create inconsistent state.
- Refresh token rotation remains safe.
- OTP verification respects one-time-use rules.
- Session state remains consistent.
- Shared resources are accessed safely.
- No goroutines leak.
- No requests wait indefinitely.

Run:

`go test -race ./...`

Do not suppress race detector failures.

Do not use arbitrary `time.Sleep()` to coordinate concurrent tests.

Prefer synchronization primitives, channels, barriers, controlled clocks, and explicit timeouts.

---

# 14. APPLICATION CONFIGURATION AND LIFECYCLE TESTING

## 14.1 Environment Configuration

- [ ] Missing environment variables.
- [ ] Invalid database URL.
- [ ] Invalid RabbitMQ URL.
- [ ] Missing JWT secret.
- [ ] Invalid JWT configuration.
- [ ] Invalid SMTP configuration.
- [ ] Invalid port configuration.
- [ ] Invalid duration values.
- [ ] Invalid security parameters.
- [ ] Valid configuration loading.
- [ ] Default values, where supported.

## 14.2 Application Startup

- [ ] Successful application startup.
- [ ] Database connection failure.
- [ ] RabbitMQ connection failure.
- [ ] Invalid configuration.
- [ ] Migration failure.
- [ ] Partial initialization failure.
- [ ] Correct resource cleanup after startup failure.

## 14.3 Application Shutdown

- [ ] Graceful HTTP server shutdown.
- [ ] Database connection closure.
- [ ] RabbitMQ connection closure.
- [ ] Context cancellation.
- [ ] Worker shutdown, when implemented.
- [ ] No goroutine leaks.
- [ ] Shutdown timeout handling.

---

# 15. PERFORMANCE AND RELIABILITY TESTING

Use Go benchmarks and controlled load tests.

Do not introduce arbitrary timing assertions into ordinary unit tests.

## 15.1 Benchmarks

Where applicable, benchmark:

- [ ] Password hashing.
- [ ] Password verification.
- [ ] JWT generation.
- [ ] JWT validation.
- [ ] OTP generation.
- [ ] OTP verification.
- [ ] Important database queries.
- [ ] Authentication service operations.

Use:

`go test -bench=. -benchmem ./...`

## 15.2 Load and Reliability Testing

Evaluate:

- [ ] Login throughput.
- [ ] Signup throughput.
- [ ] OTP verification throughput.
- [ ] Database query performance.
- [ ] RabbitMQ RPC latency.
- [ ] Concurrent login load.
- [ ] Concurrent refresh load.
- [ ] Resource consumption.
- [ ] Connection pool behavior.
- [ ] Failure recovery under load.

Document test environment, sample size, and methodology.

Do not claim production performance guarantees from small local benchmarks.

---

# 16. TEST FILE ORGANIZATION

Follow standard Go testing conventions.

Prefer keeping tests alongside the implementation they validate.

Example:

```text
authentication/
│
├── internal/
│   ├── auth/
│   │   ├── service.go
│   │   ├── handler.go
│   │   ├── repository.go
│   │   ├── service_test.go
│   │   ├── handler_test.go
│   │   └── repository_test.go
│   │
│   ├── jwt/
│   │   ├── service.go
│   │   └── service_test.go
│   │
│   ├── emailOtp/
│   │   ├── service.go
│   │   └── service_test.go
│   │
│   ├── security/
│   │   ├── password.go
│   │   └── password_test.go
│   │
│   └── messaging/
│       ├── rabbitmq.go
│       └── rabbitmq_test.go
│
├── middleware/
│   ├── auth.go
│   └── auth_test.go
│
├── tests/
│   ├── integration/
│   ├── e2e/
│   └── testdata/
│
├── .github/
│   └── workflows/
│       └── tests.yml
│
└── Testing.md

This structure is illustrative.

Adapt it to the actual repository structure.

Do not move production files unnecessarily.

17. TEST EXECUTION ORDER
Implement tests in the following order to minimize dependency-related problems.
Phase	Testing Area	Status
1	Security / Password	[ ] Pending
2	JWT Service	[ ] Pending
3	OTP Service	[ ] Pending
4	Repository / PostgreSQL	[ ] Pending
5	Authentication Service	[ ] Pending
6	RabbitMQ	[ ] Pending
7	HTTP Handlers	[ ] Pending
8	Middleware	[ ] Pending
9	End-to-End Integration	[ ] Pending
10	Concurrency / Race Detection	[ ] Pending
11	Configuration / Lifecycle	[ ] Pending
12	Performance / Reliability	[ ] Pending
13	GitHub Actions CI	[ ] Pending
14	Final Automated Review	[ ] Pending

Update the status only after the corresponding tests have actually been implemented and executed.

18. AUTOMATED REVIEW AND COVERAGE GAP ANALYSIS
THIS SECTION IS MANDATORY. DO NOT SKIP IT.

Writing tests is only the first stage.

After implementing the initial test suite, perform a complete review.
18.1 First Review — Implementation Coverage
Inspect every production package and identify:
- Public functions without tests.
- Important private functions without direct or indirect coverage.
- Untested branches.
- Untested error paths.
- Untested validation paths.
- Missing boundary conditions.
- Missing database failure scenarios.
- Missing RabbitMQ failure scenarios.
- Missing authentication workflow scenarios.
- Missing security regression tests.
Compare actual implementation against every applicable requirement in this document.
18.2 Second Review — Assertion Quality
Review every test.

Verify:
- Assertions actually validate expected behavior.
- Returned values are checked.
- Errors are checked correctly.
- Database state is verified where necessary.
- Mock expectations are correct.
- HTTP status codes are validated.
- Response bodies are validated.
- Sensitive data is checked for accidental exposure.
- Failure cases validate the correct error.
- Tests do not pass accidentally because of incomplete assertions.
Replace weak tests with meaningful tests.
18.3 Third Review — Test Isolation and Determinism
Inspect for:
- Shared mutable state.
- Global variable contamination.
- Test execution order dependencies.
- Hardcoded timestamps.
- Flaky concurrency tests.
- Uncontrolled goroutines.
- Resource leaks.
- Missing cleanup.
- Arbitrary sleeps.
- External network dependencies.
- Dependence on existing local infrastructure.
Fix all identified issues.
18.4 Fourth Review — Coverage Analysis
Run:
go test -cover ./...


Generate a coverage profile:
go test -coverprofile=coverage.out ./...


Generate a function-level coverage report:
go tool cover -func=coverage.out


Inspect uncovered sections.

Prioritize:
1. Authentication business logic.
2. JWT and session management.
3. OTP verification.
4. Password security.
5. Repository error handling.
6. RabbitMQ request/reply behavior.
7. Middleware security.
8. HTTP error handling.
Coverage targets:
Area	Target
Overall	80–85%+
Security	90%+
JWT	90%+
Critical authentication flows	90%+

These are quality targets, not permission to write meaningless tests.

Do not exclude difficult code merely to inflate coverage.

Do not change production behavior to achieve arbitrary percentages.

Document any justified uncovered code.
18.5 Fifth Review — Race and Reliability
Run:
go test -race ./...


Investigate and fix every genuine data race.

Run the test suite repeatedly to identify flaky behavior.

Where practical, use:
go test -count=10 ./...


Do not hide intermittent failures by increasing timeouts without identifying their cause.
18.6 Sixth Review — Final Test Completeness
Repeat the complete review process after implementing fixes.

The agent must continue the following cycle:
Inspect Production Code
        |
        v
Identify Missing Tests
        |
        v
Implement Tests
        |
        v
Execute Test Suite
        |
        v
Analyze Failures
        |
        v
Review Coverage
        |
        v
Review Assertions
        |
        v
Review Isolation and Race Conditions
        |
        v
Fix Gaps and Production Bugs
        |
        v
Run Complete Test Suite Again
        |
        v
Repeat Until Definition of Done


Do not declare completion while known critical gaps remain.

19. GITHUB ACTIONS AUTOMATION
After implementing and reviewing the test suite, create:

.github/workflows/tests.yml

The workflow must automate testing on:
- Push events.
- Pull request events.
Use the actual Go version declared by the repository's go.mod.

Do not blindly hardcode a different Go version.
19.1 Required CI Checks
The workflow should include appropriate jobs or steps for:
1. Checkout repository.
2. Install the required Go version.
3. Configure Go module caching.
4. Download dependencies.
5. Verify dependency consistency.
6. Run formatting checks.
7. Run go vet.
8. Run unit tests.
9. Run integration tests with PostgreSQL.
10. Run integration tests with RabbitMQ.
11. Run race detection.
12. Generate coverage reports.
13. Enforce the agreed coverage threshold.
14. Upload coverage artifacts when appropriate.
19.2 CI Infrastructure Requirements
If integration tests require PostgreSQL or RabbitMQ:
- Configure GitHub Actions service containers.
- Use isolated test databases.
- Configure health checks.
- Use test-only credentials.
- Never expose production secrets.
- Ensure services are ready before tests start.
- Ensure integration tests can be identified and executed independently.
If the project requires additional infrastructure, document it and configure it explicitly.

Do not make CI depend on a developer's local Docker environment.
19.3 Workflow Quality Rules
The workflow must:
- Use supported GitHub Actions versions.
- Use least-privilege permissions.
- Avoid unnecessary secrets.
- Avoid printing credentials.
- Use dependency caching appropriately.
- Fail when required checks fail.
- Preserve useful logs.
- Be maintainable and readable.
Do not create a workflow that reports success while silently skipping the actual tests.

If a category is intentionally excluded from CI, document the exact reason and provide an alternative execution method.

20. USEFUL GO TESTING COMMANDS
Run all tests
go test ./...


Verbose execution
go test -v ./...


Run specific package tests
go test ./internal/auth/...


Run tests with coverage
go test -cover ./...


Generate coverage profile
go test -coverprofile=coverage.out ./...


Generate HTML coverage report
go tool cover -html=coverage.out


Function-level coverage
go tool cover -func=coverage.out


Race detection
go test -race ./...


Repeated execution
go test -count=10 ./...


Run benchmarks
go test -bench=. -benchmem ./...


Static analysis
go vet ./...


Formatting verification
gofmt -l .


Clear test cache
go clean -testcache


Run commands appropriate to the actual repository and environment.

21. DEFINITION OF DONE
The testing task is complete ONLY when all applicable conditions below are satisfied.
Test Implementation
- [ ] All implemented production packages have been inspected.
- [ ] All applicable unit tests are implemented.
- [ ] All critical service methods are tested.
- [ ] All repository methods are tested.
- [ ] PostgreSQL integration tests are implemented.
- [ ] RabbitMQ integration tests are implemented.
- [ ] HTTP handlers are tested.
- [ ] Middleware is tested.
- [ ] JWT and session functionality is tested.
- [ ] OTP functionality is tested.
- [ ] Security functionality is tested.
- [ ] Critical authentication workflows are tested.
- [ ] Concurrency scenarios are tested.
- [ ] Configuration and lifecycle behavior is tested.
- [ ] Applicable performance benchmarks are implemented.
Test Quality
- [ ] No meaningless tests.
- [ ] No missing critical assertions.
- [ ] No unnecessary mocking.
- [ ] No test execution order dependencies.
- [ ] No arbitrary sleep-based synchronization.
- [ ] No uncontrolled goroutine leaks.
- [ ] Test data is deterministic.
- [ ] Resources are properly cleaned up.
- [ ] Failure paths are tested.
- [ ] Boundary conditions are tested.
Execution and Reliability
- [ ] go test ./... passes.
- [ ] go vet ./... passes.
- [ ] Formatting checks pass.
- [ ] Integration tests pass with real infrastructure.
- [ ] go test -race ./... passes.
- [ ] Repeated test execution has been reviewed.
- [ ] Coverage report has been reviewed.
- [ ] Critical coverage targets are satisfied or justified exceptions are documented.
- [ ] Production bugs discovered during testing are fixed and regression-tested.
CI/CD
- [ ] .github/workflows/tests.yml exists.
- [ ] Workflow uses the repository's supported Go version.
- [ ] PostgreSQL integration infrastructure is configured.
- [ ] RabbitMQ integration infrastructure is configured.
- [ ] Unit tests execute automatically.
- [ ] Integration tests execute automatically.
- [ ] Race detection executes automatically.
- [ ] Coverage is generated and checked.
- [ ] CI fails correctly when a required check fails.
- [ ] Workflow configuration has been reviewed.
Final Review
- [ ] Every applicable checklist item has been reviewed.
- [ ] Every remaining gap is documented.
- [ ] No critical test category has been silently skipped.
- [ ] Final complete test suite has been executed after the last code change.
- [ ] Final GitHub Actions workflow has been reviewed.

22. FINAL AGENT REPORT
After completing the entire task, provide a concise but detailed engineering report.

The report MUST contain:
22.1 Implementation Summary
- Number of test files created.
- Number of test files modified.
- Packages covered.
- Testing categories implemented.
- Production files modified, with reasons.
- New testing dependencies introduced, with reasons.
22.2 Test Execution Results
Provide actual results for:
Check	Result
Unit Tests	PASS / FAIL
PostgreSQL Integration Tests	PASS / FAIL / BLOCKED
RabbitMQ Integration Tests	PASS / FAIL / BLOCKED
HTTP Handler Tests	PASS / FAIL
Middleware Tests	PASS / FAIL
E2E Tests	PASS / FAIL / N/A
Race Detection	PASS / FAIL
Go Vet	PASS / FAIL
Formatting	PASS / FAIL
Coverage	Actual percentage
GitHub Actions Workflow	CREATED / VERIFIED

Never report a check as PASS if it was not actually executed.
22.3 Coverage Report
Provide:
- Overall coverage percentage.
- Security package coverage.
- JWT package coverage.
- Authentication package coverage.
- Critical uncovered functions.
- Justified exclusions.
22.4 Remaining Issues
Clearly list:
- Unimplemented features.
- Tests blocked by infrastructure.
- Missing integration capabilities.
- Uncovered critical paths.
- Known technical limitations.
Do not conceal incomplete work.
22.5 Final File Changes
List all created and modified files, including:
- Test files.
- Test helpers.
- Test fixtures.
- Test configuration.
- GitHub Actions workflow.
- Updated Testing.md.

23. FINAL EXECUTION DIRECTIVE
Follow this exact priority:
1. Inspect the complete existing codebase.
2. Understand the actual architecture and dependencies.
3. Map every production component to its required tests.
4. Identify missing test infrastructure and dependencies.
5. Implement unit tests.
6. Implement repository and database integration tests.
7. Implement RabbitMQ integration tests.
8. Implement service and authentication workflow tests.
9. Implement HTTP handler and middleware tests.
10. Implement security, concurrency, and lifecycle tests.
11. Implement applicable E2E tests and benchmarks.
12. Execute the complete test suite.
13. Analyze failures and fix genuine defects.
14. Generate and inspect coverage reports.
15. Perform repeated automated test quality reviews.
16. Identify and close remaining coverage gaps.
17. Repeat execution and review until all applicable requirements are satisfied.
18. Create .github/workflows/tests.yml.
19. Validate the workflow configuration and its commands.
20. Execute the final complete test suite.
21. Update this document's checklist.
22. Produce the final engineering report.
IMPORTANT: Do not stop after implementing the first batch of tests. Do not stop after achieving a coverage percentage. Do not stop after making the tests pass once.

Your responsibility is to deliver a reliable, deterministic, maintainable, security-focused, production-grade testing system.

The final result must demonstrate that the Authentication Microservice has been tested meaningfully across its actual architecture, failure modes, security boundaries, and critical workflows.

Testing is complete only when implementation, execution, review, coverage analysis, and CI automation are all complete.
```
