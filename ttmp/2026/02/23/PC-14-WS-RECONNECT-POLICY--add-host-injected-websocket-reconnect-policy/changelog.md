# Changelog

## 2026-02-23

- Initial workspace created


- Added host-injected reconnect policy types in confirm-runtime host adapter contract.
- Implemented reconnect scheduling/cancel behavior in `ConfirmWsManager`.
- Wired policy pass-through in `createConfirmRuntime` and configured exponential-ish backoff policy in inventory host.
- Added reconnect behavior unit tests (reconnect, no policy no reconnect, disconnect cancels timer).
- Validation:
  - `npx vitest run packages/confirm-runtime/src/ws/confirmWsManager.test.ts` (pass)
  - Full repository `npm run typecheck` still fails due pre-existing React typing + unrelated baseline issues outside this ticket scope.
- Code commit: `5b373ac` (go-go-os).
