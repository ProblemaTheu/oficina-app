ALTER TABLE "ordens_servico" DROP COLUMN IF EXISTS "cancelado_em";

DELETE FROM "status_ordens" WHERE "nome_status" = 'cancelada';
