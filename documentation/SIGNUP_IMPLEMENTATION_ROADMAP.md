AdOrbit Authentication Service — Signup Roadmap

Phase 1: Signup Implementation

Task 1 — Check Existing Email

* Auth DB mein credentials table se email check karna.
* Agar email already registered hai, toh appropriate error return karna.
* Agar email nahi hai, toh signup continue karna.
* Database errors ko properly handle karna.

Testing:

* Existing email → reject.
* New email → proceed.
* Database error → appropriate error.

Task 2 — Generate User ID

* Signup ke liye UUID generate karna.
* Isi UUID ko credentials, Users service aur session creation mein use karna.

Testing:

* Valid UUID generate ho.
* Ek signup ke sabhi operations mein same UUID use ho.

Task 3 — Hash Password

* Password ko Argon2id ya bcrypt se hash karna.
* Plain-text password ko database mein store nahi karna.
* Existing project conventions ke according password hashing implement karna.

Testing:

* Password hash generate ho.
* Stored password original password ke equal na ho.
* Password verification successfully work kare.

Task 4 — Save Credentials

* Auth DB ki credentials table mein user ID, email aur password hash save karna.
* Email uniqueness database constraint se enforce karna.
* Duplicate email constraint violation ko appropriate application error mein map karna.

Testing:

* Credentials successfully save hon.
* Duplicate email reject ho.
* Database errors properly handle hon.

Task 5 — Create User Through gRPC

* Auth service se Users service ka CreateUser RPC call karna.
* Same UUID, full name, email aur authorized role pass karna.
* Users service mein user profile create karna.
* gRPC errors ko appropriate application errors mein map karna.
* Duplicate user ID/email jaise conflicts handle karna.

Role Security:

* Public signup mein role server-side determine karna.
* Public users ko admin role assign karne ki permission nahi deni.
* Publisher registration ko required authorization rules ke according handle karna.

Testing:

* Users service mein profile successfully create ho.
* Correct UUID aur user details save hon.
* Duplicate user aur gRPC failure properly handle hon.

Task 6 — Create Session

* Existing JWT/session service ko integrate karna.
* Successful user creation ke baad session create karna.
* Session DB mein access token hash, refresh token hash, expiry aur existing schema ke required metadata save karna.
* Session creation failure ko properly handle karna.

Testing:

* Session successfully create aur persist ho.
* Correct user ID session se associated ho.
* Session database failure properly handle ho.

Task 7 — Return Signup Response

* User details aur access/refresh tokens return karna.
* Response DTO aur existing API conventions follow karna.
* Internal errors, password hashes aur sensitive database details response mein expose nahi karna.

Testing:

* Successful signup par correct response mile.
* Response mein expected fields hon.
* Password ya password hash response mein na ho.

⸻

Phase 2: Failure Handling & Reliability

Task 8 — Handle Concurrent Signup

* Credentials email uniqueness constraint ko final protection banana.
* Concurrent requests mein duplicate account creation prevent karna.
* Database uniqueness errors ko consistent API errors mein map karna.

Testing:

* Same email se simultaneous requests bhejna.
* Sirf ek credentials record create ho.
* Duplicate request ko appropriate error mile.

Task 9 — Handle Users gRPC Failures

* Users service unavailable hone par failure handle karna.
* gRPC deadline aur timeout configure karna.
* Retry sirf appropriate transient failures par karna.
* CreateUser operation ko idempotent banana, taaki retry se duplicate profile na bane.
* Unknown outcome ko definitive failure assume na karna.

Testing:

* Users service unavailable.
* gRPC timeout.
* User create hone ke baad response timeout.
* Retry par duplicate profile create na ho.

Task 10 — Handle Partial Signup

* Credentials save hone ke baad Users service fail hone ka case handle karna.
* User profile create hone ke baad session creation fail hone ka case handle karna.
* Safe compensation, retry ya reconciliation strategy define karna.
* Unknown gRPC outcome mein blindly credentials delete na karna.
* Existing user ko session failure ke kaaran automatically delete na karna.

Testing:

* Har failure point par database state verify karna.
* Orphaned credentials aur inconsistent records identify karna.
* Retry/recovery behavior verify karna.

Task 11 — Improve Error Handling & Logging

* Consistent application errors return karna.
* Database, gRPC aur session failures ke useful logs rakhna.
* Passwords, access tokens, refresh tokens aur sensitive credentials logs mein expose na karna.
* Client-facing errors mein internal implementation details hide karna.

Testing:

* Expected error codes aur response bodies verify karna.
* Logs mein sensitive data expose na ho.

⸻

Phase 3: Signup Integration Testing

Task 12 — Successful Signup

* Valid signup request bhejna.
* Auth DB, Users DB aur session DB verify karna.
* Confirm karna ki sabhi records correct user ID se associated hain.

Task 13 — Duplicate Email

* Existing email se signup attempt karna.
* Appropriate conflict response verify karna.
* Additional credentials, profile ya session records create na hon.

Task 14 — Request Validation

* Missing full name.
* Invalid/missing email.
* Missing password.
* Invalid password.
* Missing/invalid role, according to the API contract.

Har case mein expected validation response verify karna.

Task 15 — Users Service Failure

* Users service stop karke signup attempt karna.
* Timeout aur failure behavior verify karna.
* Database consistency aur recovery strategy test karna.

Task 16 — Session Creation Failure

* Session creation fail karwa kar signup test karna.
* Confirm karna ki user aur credentials ki state expected hai.
* Recovery behavior verify karna.

Task 17 — Final End-to-End Verification

* Successful signup.
* Duplicate email.
* Concurrent signup.
* Users service unavailable.
* gRPC timeout.
* Session creation failure.
* Database consistency.
* Correct API responses.
* Sensitive data exposure checks.

⸻

Implementation Rules

1. Ek time par sirf ek task implement karna.
2. Har task se pehle existing code review karna.
3. Existing architecture aur coding conventions follow karna.
4. Task implement karne ke baad relevant tests run karna.
5. Test pass hone aur code verify hone ke baad hi next task par jana.
6. Unnecessary refactoring avoid karna.
7. Har task ke baad changed files aur test results verify karna.
8. Failure handling ko postpone kar sakte hain, lekin uske requirements ko ignore nahi karna.

Starting Point

Current task: Task 1 — Check Existing Email

Auth service ke existing credentials repository, database queries aur signup-related code ko review karke implementation start karna. Task 1 complete aur tested hone ke baad hi Task 2 par jana.