-- Reverte 000007. Ordem inversa da subida.
-- Falha de propósito se existir OS `cancelada` (FK de ordens_servico.status_id):
-- reverter com dados de saga em uso exige decidir o destino dessas OS antes.

DROP TABLE IF EXISTS "mensagens_processadas";
DROP TABLE IF EXISTS "outbox";
DROP TABLE IF EXISTS "sagas_os_etapas";
DROP TABLE IF EXISTS "sagas_os";

ALTER TABLE "itens_os_pecas"
  DROP CONSTRAINT IF EXISTS "chk_itens_reserva_estado",
  DROP COLUMN IF EXISTS "reserva_estado";

ALTER TABLE "pecas"
  DROP CONSTRAINT IF EXISTS "chk_pecas_reserva_valida",
  DROP COLUMN IF EXISTS "estoque_reservado";

ALTER TABLE "ordens_servico"
  DROP COLUMN IF EXISTS "motivo_cancelamento",
  DROP COLUMN IF EXISTS "cancelado_em";

UPDATE "status_ordens" SET "descricao" = 'Serviço concluído ou orçamento rejeitado pelo cliente.'
WHERE "nome_status" = 'finalizada';

DELETE FROM "status_ordens" WHERE "nome_status" = 'cancelada';
