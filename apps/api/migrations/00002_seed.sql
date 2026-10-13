-- +goose Up
INSERT INTO accounts (name) VALUES
  ('cash'),
  ('bank'),
  ('e-wallet')
ON CONFLICT (name) DO NOTHING;

INSERT INTO categories (name, keywords) VALUES
  ('makan-minum', ARRAY['makan','minum','kopi','resto','warung','nasi','ayam','bakso','jajan']),
  ('transport', ARRAY['transport','ojek','gojek','grab','bensin','parkir','kereta','bus','ongkir']),
  ('belanja', ARRAY['belanja','baju','sepatu','shopee','tokopedia','lazada','supermarket']),
  ('tagihan', ARRAY['tagihan','listrik','air','internet','wifi','pulsa','kuota','sewa']),
  ('hiburan', ARRAY['hiburan','nonton','bioskop','game','spotify','netflix','liburan']),
  ('kesehatan', ARRAY['kesehatan','obat','dokter','apotek','vitamin']),
  ('lainnya', ARRAY['lainnya'])
ON CONFLICT (name) DO NOTHING;

-- +goose Down
DELETE FROM categories WHERE name IN
  ('makan-minum','transport','belanja','tagihan','hiburan','kesehatan','lainnya');

DELETE FROM accounts WHERE name IN ('cash','bank','e-wallet');
