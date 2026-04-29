DROP SEQUENCE IF EXISTS os_numero_seq;

ALTER TABLE "historicos_status"
  DROP COLUMN IF EXISTS "observacao";

ALTER TABLE "ordens_servico"
  DROP COLUMN IF EXISTS "entregue_em",
  DROP COLUMN IF EXISTS "finalizado_em",
  DROP COLUMN IF EXISTS "iniciado_em",
  DROP COLUMN IF EXISTS "reprovado_em",
  DROP COLUMN IF EXISTS "aprovado_em",
  DROP COLUMN IF EXISTS "diagnostico",
  DROP COLUMN IF EXISTS "descricao",
  DROP COLUMN IF EXISTS "numero";
