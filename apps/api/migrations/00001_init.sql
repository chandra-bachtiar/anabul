-- +goose Up
CREATE TABLE accounts (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE categories (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  keywords TEXT[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE transactions (
  id BIGSERIAL PRIMARY KEY,
  type TEXT NOT NULL CHECK (type IN ('expense','income')),
  amount BIGINT NOT NULL CHECK (amount > 0),
  note TEXT NOT NULL DEFAULT '',
  category_id BIGINT REFERENCES categories(id),
  account_id BIGINT REFERENCES accounts(id),
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  month_start DATE NOT NULL,
  source TEXT NOT NULL DEFAULT 'manual',
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX transactions_month_category_idx
  ON transactions (month_start, category_id)
  WHERE deleted_at IS NULL;

CREATE TABLE budgets (
  id BIGSERIAL PRIMARY KEY,
  month_start DATE NOT NULL,
  category_id BIGINT NOT NULL REFERENCES categories(id),
  amount BIGINT NOT NULL CHECK (amount >= 0),
  UNIQUE (month_start, category_id)
);

CREATE TABLE recurring_needs (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  amount BIGINT NOT NULL CHECK (amount >= 0),
  due_day SMALLINT NOT NULL CHECK (due_day BETWEEN 1 AND 31),
  category_id BIGINT REFERENCES categories(id),
  active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE monthly_needs (
  id BIGSERIAL PRIMARY KEY,
  recurring_need_id BIGINT REFERENCES recurring_needs(id),
  month_start DATE NOT NULL,
  name TEXT NOT NULL,
  amount BIGINT NOT NULL,
  due_date DATE NOT NULL,
  status TEXT NOT NULL DEFAULT 'unpaid' CHECK (status IN ('unpaid','paid')),
  transaction_id BIGINT REFERENCES transactions(id),
  UNIQUE (recurring_need_id, month_start)
);

CREATE TABLE debts (
  id BIGSERIAL PRIMARY KEY,
  person TEXT NOT NULL,
  direction TEXT NOT NULL CHECK (direction IN ('hutang','piutang')),
  amount_total BIGINT NOT NULL CHECK (amount_total > 0),
  note TEXT NOT NULL DEFAULT '',
  due_date DATE,
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','settled')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE debt_payments (
  id BIGSERIAL PRIMARY KEY,
  debt_id BIGINT NOT NULL REFERENCES debts(id),
  amount BIGINT NOT NULL CHECK (amount > 0),
  paid_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  transaction_id BIGINT REFERENCES transactions(id)
);

CREATE TABLE month_closings (
  month_start DATE PRIMARY KEY,
  closed_at TIMESTAMPTZ,
  reopened_at TIMESTAMPTZ,
  snapshot JSONB NOT NULL
);

CREATE TABLE pending_actions (
  id TEXT PRIMARY KEY,
  tool TEXT NOT NULL,
  args JSONB NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE agent_logs (
  id BIGSERIAL PRIMARY KEY,
  channel TEXT NOT NULL,
  user_message TEXT NOT NULL,
  tool_calls JSONB NOT NULL DEFAULT '[]',
  reply TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE chat_messages (
  id BIGSERIAL PRIMARY KEY,
  channel TEXT NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE chat_messages;
DROP TABLE agent_logs;
DROP TABLE pending_actions;
DROP TABLE month_closings;
DROP TABLE debt_payments;
DROP TABLE debts;
DROP TABLE monthly_needs;
DROP TABLE recurring_needs;
DROP TABLE budgets;
DROP TABLE transactions;
DROP TABLE categories;
DROP TABLE accounts;
