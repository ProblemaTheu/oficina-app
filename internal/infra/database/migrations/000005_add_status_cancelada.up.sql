INSERT INTO "status_ordens" ("id", "nome_status", "descricao") VALUES
  ('660e8400-e29b-41d4-a716-446655440007', 'cancelada', 'OS cancelada antes da conclusão do serviço.')
ON CONFLICT DO NOTHING;

ALTER TABLE "ordens_servico"
  ADD COLUMN IF NOT EXISTS "cancelado_em" timestamp;

COMMENT ON COLUMN "ordens_servico"."cancelado_em" IS 'Timestamp de quando a OS foi cancelada.';
