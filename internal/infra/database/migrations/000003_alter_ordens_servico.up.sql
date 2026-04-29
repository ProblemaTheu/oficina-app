ALTER TABLE "ordens_servico"
  ADD COLUMN "numero"        varchar(20)    UNIQUE,
  ADD COLUMN "descricao"     text,
  ADD COLUMN "diagnostico"   text,
  ADD COLUMN "aprovado_em"   timestamp,
  ADD COLUMN "reprovado_em"  timestamp,
  ADD COLUMN "iniciado_em"   timestamp,
  ADD COLUMN "finalizado_em" timestamp,
  ADD COLUMN "entregue_em"   timestamp;

COMMENT ON COLUMN "ordens_servico"."numero"        IS 'Número legível da OS no formato OS-YYYY-NNNNN (ex: OS-2024-00001).';
COMMENT ON COLUMN "ordens_servico"."descricao"     IS 'Problema relatado pelo cliente na abertura da OS.';
COMMENT ON COLUMN "ordens_servico"."diagnostico"   IS 'Diagnóstico preenchido pelo mecânico. Obrigatório na transição para aguardando_aprovacao.';
COMMENT ON COLUMN "ordens_servico"."aprovado_em"   IS 'Timestamp de quando o cliente aprovou o orçamento.';
COMMENT ON COLUMN "ordens_servico"."reprovado_em"  IS 'Timestamp de quando o cliente rejeitou o orçamento.';
COMMENT ON COLUMN "ordens_servico"."iniciado_em"   IS 'Timestamp de quando a execução começou (status → em_execucao). Usado no cálculo de tempo médio.';
COMMENT ON COLUMN "ordens_servico"."finalizado_em" IS 'Timestamp de quando o serviço foi concluído (status → finalizada). Usado no cálculo de tempo médio.';
COMMENT ON COLUMN "ordens_servico"."entregue_em"   IS 'Timestamp de quando o veículo foi entregue ao cliente.';

ALTER TABLE "historicos_status"
  ADD COLUMN "observacao" text;

COMMENT ON COLUMN "historicos_status"."observacao" IS 'Anotação livre registrada na transição de status (ex: diagnóstico do mecânico, motivo de rejeição).';

CREATE SEQUENCE IF NOT EXISTS os_numero_seq START 1;
