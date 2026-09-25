-- Sample drivers near Thiruvananthapuram
INSERT INTO drivers (id, name, phone, status, current_lat, current_lng, capacity)
VALUES
  ('d1', 'Rajan Kumar',  '9876543210', 'available', 8.5241, 76.9366, 3),
  ('d2', 'Suresh Nair',  '9876543211', 'available', 8.5300, 76.9400, 2),
  ('d3', 'Arun Pillai',  '9876543212', 'available', 8.5150, 76.9200, 4),
  ('d4', 'Priya Menon',  '9876543213', 'available', 8.5450, 76.9500, 3)
ON CONFLICT DO NOTHING;
