-- Reverte 000005. Ordem inversa da subida.

DROP INDEX IF EXISTS "ux_itens_os_servicos";
DROP INDEX IF EXISTS "ux_itens_os_pecas";

ALTER TABLE "ordens_servico" DROP CONSTRAINT IF EXISTS "chk_os_valor_nao_negativo";
ALTER TABLE "itens_os_pecas" DROP CONSTRAINT IF EXISTS "chk_itens_quantidade_positiva";
ALTER TABLE "pecas"
  DROP CONSTRAINT IF EXISTS "chk_pecas_preco_nao_negativo",
  DROP CONSTRAINT IF EXISTS "chk_pecas_estoque_nao_negativo";

-- timestamptz → timestamp: converte de volta para UTC antes de descartar o fuso.
ALTER TABLE "historicos_status"
  ALTER COLUMN "alterado_em" TYPE timestamp USING "alterado_em" AT TIME ZONE 'UTC';

ALTER TABLE "ordens_servico"
  ALTER COLUMN "criado_em"     TYPE timestamp USING "criado_em"     AT TIME ZONE 'UTC',
  ALTER COLUMN "atualizado_em" TYPE timestamp USING "atualizado_em" AT TIME ZONE 'UTC',
  ALTER COLUMN "aprovado_em"   TYPE timestamp USING "aprovado_em"   AT TIME ZONE 'UTC',
  ALTER COLUMN "reprovado_em"  TYPE timestamp USING "reprovado_em"  AT TIME ZONE 'UTC',
  ALTER COLUMN "iniciado_em"   TYPE timestamp USING "iniciado_em"   AT TIME ZONE 'UTC',
  ALTER COLUMN "finalizado_em" TYPE timestamp USING "finalizado_em" AT TIME ZONE 'UTC',
  ALTER COLUMN "entregue_em"   TYPE timestamp USING "entregue_em"   AT TIME ZONE 'UTC';

ALTER TABLE "clientes"
  ALTER COLUMN "criado_em"     TYPE timestamp USING "criado_em"     AT TIME ZONE 'UTC',
  ALTER COLUMN "atualizado_em" TYPE timestamp USING "atualizado_em" AT TIME ZONE 'UTC';

DROP INDEX IF EXISTS "ix_os_status_criado";

DROP INDEX IF EXISTS "ix_usuarios_papel";
DROP INDEX IF EXISTS "ix_historicos_os";
DROP INDEX IF EXISTS "ix_itens_os_pecas_peca";
DROP INDEX IF EXISTS "ix_itens_os_pecas_os";
DROP INDEX IF EXISTS "ix_itens_os_servicos_servico";
DROP INDEX IF EXISTS "ix_itens_os_servicos_os";
DROP INDEX IF EXISTS "ix_os_responsavel";
DROP INDEX IF EXISTS "ix_os_veiculo";
DROP INDEX IF EXISTS "ix_os_cliente";
DROP INDEX IF EXISTS "ix_veiculos_cliente";

DROP INDEX IF EXISTS "ux_clientes_cpf_digitos";
ALTER TABLE "clientes" DROP COLUMN IF EXISTS "cpf_cnpj_digitos";

ALTER TABLE "clientes" DROP CONSTRAINT IF EXISTS "chk_clientes_status";
ALTER TABLE "clientes" DROP COLUMN IF EXISTS "status";
