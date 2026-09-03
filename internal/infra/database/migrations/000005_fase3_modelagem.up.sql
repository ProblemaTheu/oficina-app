-- Fase 3 — revisão do modelo relacional (F3-5.1).
-- Aditiva, exceto pelos ALTER COLUMN TYPE, que reescrevem a tabela.

-- ── 1. Status do cliente ────────────────────────────────────────────────────
-- A Lambda de autenticação precisa consultar a existência E o status:
-- cliente inativo recebe 403, não token.
ALTER TABLE "clientes"
  ADD COLUMN "status" varchar(20) NOT NULL DEFAULT 'ativo';

ALTER TABLE "clientes"
  ADD CONSTRAINT "chk_clientes_status"
  CHECK ("status" IN ('ativo', 'inativo', 'bloqueado'));

COMMENT ON COLUMN "clientes"."status" IS
  'Situação do cadastro. Somente clientes ativos obtêm token de autenticação.';

-- ── 2. CPF/CNPJ normalizado ─────────────────────────────────────────────────
-- O login por CPF consulta por dígitos, sem máscara. Coluna GERADA: o banco
-- mantém sincronizada com cpf_cnpj, então é impossível divergir.
ALTER TABLE "clientes"
  ADD COLUMN "cpf_cnpj_digitos" varchar(14)
  GENERATED ALWAYS AS (regexp_replace("cpf_cnpj", '[^0-9]', '', 'g')) STORED;

CREATE UNIQUE INDEX "ux_clientes_cpf_digitos" ON "clientes" ("cpf_cnpj_digitos");

COMMENT ON COLUMN "clientes"."cpf_cnpj_digitos" IS
  'CPF/CNPJ somente dígitos, derivado de cpf_cnpj. Usado no login por CPF.';

-- ── 3. Índices nas foreign keys ─────────────────────────────────────────────
-- No PostgreSQL, FK NÃO cria índice automaticamente (ao contrário do MySQL).
-- Sem eles, todo JOIN e todo DELETE em cascata varre a tabela inteira.
CREATE INDEX "ix_veiculos_cliente"          ON "veiculos" ("cliente_id");
CREATE INDEX "ix_os_cliente"                ON "ordens_servico" ("cliente_id");
CREATE INDEX "ix_os_veiculo"                ON "ordens_servico" ("veiculo_id");
CREATE INDEX "ix_os_responsavel"            ON "ordens_servico" ("usuario_responsavel_id");
CREATE INDEX "ix_itens_os_servicos_os"      ON "itens_os_servicos" ("os_id");
CREATE INDEX "ix_itens_os_servicos_servico" ON "itens_os_servicos" ("servico_id");
CREATE INDEX "ix_itens_os_pecas_os"         ON "itens_os_pecas" ("os_id");
CREATE INDEX "ix_itens_os_pecas_peca"       ON "itens_os_pecas" ("peca_id");
CREATE INDEX "ix_historicos_os"             ON "historicos_status" ("os_id", "alterado_em" DESC);
CREATE INDEX "ix_usuarios_papel"            ON "usuarios" ("papel_id");

-- ── 4. Índice da listagem de OS ─────────────────────────────────────────────
-- A query mais executada da API ordena por prioridade de status e criado_em.
CREATE INDEX "ix_os_status_criado" ON "ordens_servico" ("status_id", "criado_em" ASC);

-- ── 5. Timezone ─────────────────────────────────────────────────────────────
-- O pod e o RDS rodam em UTC, mas "volume DIÁRIO de OS" é em America/Sao_Paulo:
-- 3 h de deslocamento. timestamptz guarda o instante absoluto e deixa a
-- conversão para a apresentação.
ALTER TABLE "clientes"
  ALTER COLUMN "criado_em"     TYPE timestamptz USING "criado_em"     AT TIME ZONE 'UTC',
  ALTER COLUMN "atualizado_em" TYPE timestamptz USING "atualizado_em" AT TIME ZONE 'UTC';

ALTER TABLE "ordens_servico"
  ALTER COLUMN "criado_em"     TYPE timestamptz USING "criado_em"     AT TIME ZONE 'UTC',
  ALTER COLUMN "atualizado_em" TYPE timestamptz USING "atualizado_em" AT TIME ZONE 'UTC',
  ALTER COLUMN "aprovado_em"   TYPE timestamptz USING "aprovado_em"   AT TIME ZONE 'UTC',
  ALTER COLUMN "reprovado_em"  TYPE timestamptz USING "reprovado_em"  AT TIME ZONE 'UTC',
  ALTER COLUMN "iniciado_em"   TYPE timestamptz USING "iniciado_em"   AT TIME ZONE 'UTC',
  ALTER COLUMN "finalizado_em" TYPE timestamptz USING "finalizado_em" AT TIME ZONE 'UTC',
  ALTER COLUMN "entregue_em"   TYPE timestamptz USING "entregue_em"   AT TIME ZONE 'UTC';

ALTER TABLE "historicos_status"
  ALTER COLUMN "alterado_em" TYPE timestamptz USING "alterado_em" AT TIME ZONE 'UTC';

-- ── 6. Consistência de valores ──────────────────────────────────────────────
ALTER TABLE "pecas"
  ADD CONSTRAINT "chk_pecas_estoque_nao_negativo" CHECK ("estoque_atual" >= 0),
  ADD CONSTRAINT "chk_pecas_preco_nao_negativo"   CHECK ("preco" >= 0);

ALTER TABLE "itens_os_pecas"
  ADD CONSTRAINT "chk_itens_quantidade_positiva" CHECK ("quantidade" > 0);

ALTER TABLE "ordens_servico"
  ADD CONSTRAINT "chk_os_valor_nao_negativo" CHECK ("valor_total" >= 0);

-- Uma OS não lança a mesma peça/serviço duas vezes: soma na quantidade.
CREATE UNIQUE INDEX "ux_itens_os_pecas"    ON "itens_os_pecas" ("os_id", "peca_id");
CREATE UNIQUE INDEX "ux_itens_os_servicos" ON "itens_os_servicos" ("os_id", "servico_id");
