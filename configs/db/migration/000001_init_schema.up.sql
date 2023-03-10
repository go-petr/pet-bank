CREATE TABLE "accounts" (
  "id" serial PRIMARY KEY,
  "owner" varchar NOT NULL,
  "balance" numeric NOT NULL CHECK ("balance" >= 0),
  "currency" varchar NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  UNIQUE ("owner", "currency")
);

CREATE INDEX ON "accounts" ("owner");

CREATE TABLE "entries" (
  "id" bigserial PRIMARY KEY,
  "account_id" int NOT NULL,
  "amount" numeric NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON DELETE CASCADE
);

CREATE INDEX ON "entries" ("account_id");
COMMENT ON COLUMN "entries"."amount" IS 'can be negative or positive';

CREATE TABLE "transfers" (
  "id" bigserial PRIMARY KEY,
  "from_account_id" int NOT NULL,
  "to_account_id" int NOT NULL,
  "amount" numeric NOT NULL CHECK ("amount" > 0),
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  FOREIGN KEY ("from_account_id") REFERENCES "accounts" ("id") ON DELETE CASCADE,
  FOREIGN KEY ("to_account_id") REFERENCES "accounts" ("id") ON DELETE CASCADE
);

CREATE INDEX ON "transfers" ("from_account_id");
CREATE INDEX ON "transfers" ("to_account_id");
CREATE INDEX ON "transfers" ("from_account_id", "to_account_id");
