-- Remove os dados de demonstração. Os veículos saem antes dos clientes
-- por causa da foreign key.

DELETE FROM "veiculos" WHERE "id" IN (
  '880e8400-e29b-41d4-a716-446655440001',
  '880e8400-e29b-41d4-a716-446655440002'
);

DELETE FROM "clientes" WHERE "id" IN (
  '770e8400-e29b-41d4-a716-446655440001',
  '770e8400-e29b-41d4-a716-446655440002'
);
