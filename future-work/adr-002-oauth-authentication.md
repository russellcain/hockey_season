# ADR-002: Migrate from Team Codes to Google OAuth

## Status
Proposed (not yet implemented)

## Context

The current authentication scheme uses pre-shared HMAC-verified team codes (`TEAM-XXXX`). The commissioner generates one code per team, each drafter enters their code on first load, and the server exchanges it for a 24-hour signed session token.

This is appropriate for a small closed group of friends but has limitations:
- Codes can be accidentally shared, allowing someone to impersonate a drafter
- No cross-season identity — each season requires new codes
- No audit trail linking picks to real identities
- Can't support multi-device sessions without re-entering the code

## Decision

When any of the following triggers apply, migrate to Google OAuth:

- The pool grows beyond a single trusted group (open leagues, strangers)
- Multi-season player history tracking is needed
- Multiple devices per drafter are a hard requirement

## Migration Path

### 1. Schema change

```sql
ALTER TABLE fantasy_teams ADD COLUMN google_sub TEXT UNIQUE;
```

`google_sub` is the stable subject identifier from Google's ID token (`sub` claim). It never changes even if the user's email changes.

### 2. New endpoint

```
POST /api/auth/google
  Body:    { "idToken": "<Google ID token from frontend sign-in>" }
  Action:  verify token with Google's tokeninfo API or public keys
           look up fantasy_teams WHERE google_sub = token.sub
  Returns: { "token": "<HMAC session token>", "team": { ... } }
           or 401 if the google_sub is not mapped to any team
```

The session token format is unchanged — only the lookup mechanism changes.

### 3. Commissioner setup

Instead of generating codes, the commissioner collects each drafter's Google email, resolves their `sub` via a one-time Google sign-in lookup, and inserts it into `fantasy_teams.google_sub`.

A small CLI helper (`cmd/setup-teams`) should handle this step.

### 4. Backward compatibility

Keep the code-based `/api/auth/join` endpoint active during the transition season so drafters can use either method. Remove it the following season once everyone has linked their Google account.

### 5. Frontend change

Add a "Sign in with Google" button on the join screen. On success, exchange the Google ID token at `/api/auth/google`. The rest of the frontend (draft room, WebSocket handling) is unchanged — it operates on the HMAC session token regardless of how it was obtained.

## Trade-offs

| | Team codes (current) | Google OAuth (future) |
|---|---|---|
| Setup friction | Low — share a code | Medium — collect Google emails |
| Identity guarantee | Weak — code is shareable | Strong — tied to Google account |
| Multi-device | Re-enter code each device | Seamless |
| Cross-season identity | None | Persistent via `google_sub` |
| External dependency | None | Google's OAuth servers |
