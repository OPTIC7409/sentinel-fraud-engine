DROP INDEX IF EXISTS idx_transactions_amount;
DROP INDEX IF EXISTS idx_transactions_merchant_category;
DROP INDEX IF EXISTS idx_transactions_timestamp;
DROP INDEX IF EXISTS idx_transactions_merchant;
DROP INDEX IF EXISTS idx_transactions_user_timestamp;
DROP TABLE IF EXISTS transactions CASCADE;
