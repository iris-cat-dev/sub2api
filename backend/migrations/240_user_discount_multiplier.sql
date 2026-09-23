ALTER TABLE users
    ADD COLUMN IF NOT EXISTS discount_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_discount_multiplier_range;

ALTER TABLE users
    ADD CONSTRAINT users_discount_multiplier_range
    CHECK (discount_multiplier > 0 AND discount_multiplier <= 1);

COMMENT ON COLUMN users.discount_multiplier IS
    'Global customer discount applied after channel/group/user-specific pricing; 1.0 means no discount';
