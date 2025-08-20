-- Migration to create base tables for DayBoard.

-- Users table stores application users. OAuth tokens are stored in a
-- separate table to maintain referential integrity and allow multiple
-- tokens per user.
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- OAuth tokens store encrypted access and refresh tokens for each
-- provider. A user may have multiple tokens if connected to
-- Google, Microsoft, and Plaid. Encryption should be handled
-- application-side before inserting into this table.
CREATE TABLE IF NOT EXISTS oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    access_token_enc BYTEA NOT NULL,
    refresh_token_enc BYTEA,
    scopes TEXT[] NOT NULL,
    expiry TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Calendar events store normalized events for a user's agenda. Events
-- are fetched from external providers and cached here to avoid
-- hitting provider APIs on each request.
CREATE TABLE IF NOT EXISTS calendar_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    ext_id TEXT NOT NULL,
    start_ts TIMESTAMPTZ NOT NULL,
    end_ts TIMESTAMPTZ NOT NULL,
    title TEXT,
    join_url TEXT,
    location TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, source, ext_id)
);

-- Subscriptions table stores recurring charges discovered via Plaid
-- transactions or entered manually by the user. Cadence is stored in
-- days for simplicity (e.g., 30 for monthly, 7 for weekly).
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    merchant TEXT NOT NULL,
    amount_cents INT NOT NULL,
    cadence_days INT NOT NULL,
    next_due DATE,
    source TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Transactions table stores raw transaction data from Plaid or CSV
-- imports. This table may be used to re-run recurring detection when
-- algorithms improve. Raw data is stored in JSONB for flexibility.
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    ext_id TEXT,
    txn_date DATE NOT NULL,
    merchant TEXT,
    amount_cents INT NOT NULL,
    category TEXT,
    raw JSONB
);

-- Profiles table stores per-user settings used by the estimator. Each
-- user has a single profile row.
CREATE TABLE IF NOT EXISTS profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    home_addr TEXT,
    office_addr TEXT,
    city TEXT,
    state TEXT,
    hourly_cents INT,
    hours_per_week INT,
    stipend_cents INT,
    pay_freq TEXT,
    start_date DATE,
    in_office_days INT DEFAULT 3,
    food_cost_cents INT DEFAULT 1200
);

-- Federal tax tables store brackets and rates for a given year. Rates
-- are stored in basis points (bps) to avoid floating point issues.
CREATE TABLE IF NOT EXISTS tax_tables_federal (
    year INT NOT NULL,
    bracket_low INT NOT NULL,
    bracket_high INT NOT NULL,
    rate_bps INT NOT NULL,
    std_deduction_single INT NOT NULL,
    std_deduction_mfj INT NOT NULL
);

-- State tax tables store brackets and rates by state and year.
CREATE TABLE IF NOT EXISTS tax_tables_state (
    state TEXT NOT NULL,
    year INT NOT NULL,
    bracket_low INT NOT NULL,
    bracket_high INT NOT NULL,
    rate_bps INT NOT NULL,
    std_deduction_single INT NOT NULL
);

-- City cost models store base fare and per-unit costs for the commute
-- estimator. Costs are stored in cents for precision.
CREATE TABLE IF NOT EXISTS city_cost_models (
    city TEXT PRIMARY KEY,
    base_fare_cents INT NOT NULL,
    per_mile_cents INT NOT NULL,
    per_minute_cents INT NOT NULL
);


