-- Fase 4 — saga da OS (F4-1.2): estado terminal de falha, reserva de estoque,
-- estado da saga, outbox transacional e inbox idempotente. Aditiva.

-- ── 1. Status `cancelada` ───────────────────────────────────────────────────
-- Destino de uma saga compensada. Sem ele, OS rejeitada e OS concluída caíam
-- no mesmo `finalizada` e o dashboard contava compensação como conclusão.
INSERT INTO "status_ordens" ("id", "nome_status", "descricao") VALUES
  ('660e8400-e29b-41d4-a716-446655440007', 'cancelada',
   'OS cancelada pela compensação da saga (orçamento recusado, pagamento recusado ou falha na execução).')
ON CONFLICT ("nome_status") DO NOTHING;

UPDATE "status_ordens" SET "descricao" = 'Serviço concluído.' WHERE "nome_status" = 'finalizada';

ALTER TABLE "ordens_servico"
  ADD COLUMN "cancelado_em"        timestamptz,
  ADD COLUMN "motivo_cancelamento" text;

-- ── 2. Reserva de estoque ───────────────────────────────────────────────────
-- Reservar segura peças na abertura da OS sem retirá-las; a baixa só acontece
-- quando o reparo começa (pivô da saga). O CHECK barra reserva maior que o
-- estoque mesmo se o código errar.
ALTER TABLE "pecas"
  ADD COLUMN "estoque_reservado" integer NOT NULL DEFAULT 0,
  ADD CONSTRAINT "chk_pecas_reserva_valida"
    CHECK ("estoque_reservado" >= 0 AND "estoque_reservado" <= "estoque_atual");

COMMENT ON COLUMN "pecas"."estoque_reservado" IS
  'Unidades reservadas por OS em andamento. Disponível = estoque_atual - estoque_reservado.';

-- Torna Liberar/Baixar idempotentes por item. NULL = item ainda não reservado
-- (ou OS anterior à saga, cujo estoque foi deduzido pelo fluxo antigo).
ALTER TABLE "itens_os_pecas"
  ADD COLUMN "reserva_estado" varchar(10),
  ADD CONSTRAINT "chk_itens_reserva_estado"
    CHECK ("reserva_estado" IN ('reservada', 'liberada', 'baixada'));

-- ── 3. Estado da saga ───────────────────────────────────────────────────────
CREATE TABLE "sagas_os" (
  "id"            uuid PRIMARY KEY,
  "os_id"         uuid NOT NULL UNIQUE REFERENCES "ordens_servico" ("id"),
  "estado"        varchar(30) NOT NULL,
  "etapa_atual"   varchar(5)  NOT NULL,
  "etapa_falha"   varchar(5),
  "orcamento_id"  uuid,
  "pagamento_id"  uuid,
  "prazo_em"      timestamptz,
  "versao"        integer NOT NULL DEFAULT 1,
  "criado_em"     timestamptz NOT NULL DEFAULT now(),
  "atualizado_em" timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE "sagas_os" IS
  'Estado da saga orquestrada de cada OS. versao = lock otimista entre réplicas do orquestrador.';

-- Reservado para o watchdog de prazos (F4-4.6, evolução).
CREATE INDEX "ix_sagas_os_prazo" ON "sagas_os" ("prazo_em")
  WHERE "estado" NOT IN ('CONCLUIDA', 'COMPENSADA');

CREATE TABLE "sagas_os_etapas" (
  "id"          uuid PRIMARY KEY,
  "saga_id"     uuid NOT NULL REFERENCES "sagas_os" ("id"),
  "etapa"       varchar(5)  NOT NULL,
  "resultado"   varchar(12) NOT NULL
    CONSTRAINT "chk_sagas_etapas_resultado" CHECK ("resultado" IN ('ok', 'falha', 'compensada', 'ignorada')),
  "message_id"  uuid,
  "detalhe"     jsonb,
  "ocorrido_em" timestamptz NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX "ix_sagas_etapas_saga" ON "sagas_os_etapas" ("saga_id", "ocorrido_em");

-- ── 4. Outbox transacional ──────────────────────────────────────────────────
-- Gravada na MESMA transação do efeito; o relay publica no SQS.
-- `seq` define a ordem: now() é o início da transação, então várias mensagens
-- gravadas na mesma tx teriam o mesmo criado_em.
CREATE TABLE "outbox" (
  "id"            uuid PRIMARY KEY,
  "seq"           bigint GENERATED ALWAYS AS IDENTITY,
  "destino"       varchar(30) NOT NULL,
  "tipo"          varchar(40) NOT NULL,
  "envelope"      jsonb NOT NULL,
  "trace_headers" jsonb NOT NULL DEFAULT '{}',
  "criado_em"     timestamptz NOT NULL DEFAULT now(),
  "enviado_em"    timestamptz,
  "tentativas"    integer NOT NULL DEFAULT 0,
  "falhou_em"     timestamptz
);

COMMENT ON COLUMN "outbox"."falhou_em" IS
  'Preenchido quando o relay desiste (após N tentativas ou erro permanente): a linha deixa de travar as outras sagas.';

CREATE INDEX "ix_outbox_pendentes" ON "outbox" ("seq")
  WHERE "enviado_em" IS NULL AND "falhou_em" IS NULL;

-- ── 5. Inbox (consumidor idempotente) ───────────────────────────────────────
-- A deduplicação do SQS FIFO vale 5 min; depois disso, quem protege é esta tabela.
CREATE TABLE "mensagens_processadas" (
  "message_id"    uuid PRIMARY KEY,
  "tipo"          varchar(40) NOT NULL,
  "processado_em" timestamptz NOT NULL DEFAULT now()
);
