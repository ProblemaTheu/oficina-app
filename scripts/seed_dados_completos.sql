-- ============================================================
-- SEED: Dados Completos para Teste dos Endpoints
-- ============================================================
-- Script de dados mockados: clientes, veículos e ordens de serviço
-- em todos os 6 status possíveis. Requer as migrations aplicadas
--
-- Objetivo: banco de dados completamente populado para verificar
-- o funcionamento de todos os endpoints da API sem cadastrar
-- nada manualmente.
--
-- Execute APÓS as migrations (inclui 000004_seed_catalogo):
--   psql -U postgres -d tech_challenge_db -f scripts/seed_dados_completos.sql
--
-- Todos os INSERTs usam ON CONFLICT DO NOTHING — seguro re-executar.
-- ============================================================
--
-- CREDENCIAIS DE ACESSO (para POST /auth/login):
--   admin@oficina.com   → Admin@123       (administrador)
--   joao@oficina.com    → Mecanico@123    (mecanico)
--   ana@oficina.com     → Atende@123      (atendente)
--
-- UUIDs ÚTEIS PARA OS ENDPOINTS:
--   Clientes:
--     João Silva (CPF)          → 22000000-0000-0000-0000-000000000001
--     Maria Santos (CPF)        → 22000000-0000-0000-0000-000000000002
--     Carlos Oliveira (CPF)     → 22000000-0000-0000-0000-000000000003
--     Auto Peças Rápido (CNPJ)  → 22000000-0000-0000-0000-000000000004
--     Transportes Veloz (CNPJ)  → 22000000-0000-0000-0000-000000000005
--
--   Veículos:
--     VW Gol 2018 (João)        → 33000000-0000-0000-0000-000000000001
--     Chevrolet Onix 2020 (Maria) → 33000000-0000-0000-0000-000000000002
--     Toyota Corolla 2022 (Carlos) → 33000000-0000-0000-0000-000000000003
--     Fiat Uno 2015 (Auto Peças) → 33000000-0000-0000-0000-000000000004
--     Ford Ka 2019 (Transportes) → 33000000-0000-0000-0000-000000000005
--     VW T-Cross 2023 (João)    → 33000000-0000-0000-0000-000000000006
--     Honda Civic 2021 (Maria)  → 33000000-0000-0000-0000-000000000007
--     Renault Kwid 2021 (Carlos) → 33000000-0000-0000-0000-000000000008
--
--   Ordens de Serviço (por status):
--     recebida              OS-2026-00001 → 66000000-0000-0000-0000-000000000001
--     em_diagnostico        OS-2026-00002 → 66000000-0000-0000-0000-000000000002
--     aguardando_aprovacao  OS-2026-00003 → 66000000-0000-0000-0000-000000000003
--     em_execucao           OS-2026-00004 → 66000000-0000-0000-0000-000000000004
--     finalizada (concluída)OS-2026-00005 → 66000000-0000-0000-0000-000000000005
--     entregue              OS-2026-00006 → 66000000-0000-0000-0000-000000000006
--     finalizada (rejeitada)OS-2026-00007 → 66000000-0000-0000-0000-000000000007
-- ============================================================

-- ════════════════════════════════════════════════════════════
-- 1. CLIENTES
-- ════════════════════════════════════════════════════════════
-- CPF/CNPJ armazenados como dígitos (sem pontuação), conforme
-- normalização aplicada pela camada de domínio (valueobject.Document).
-- CPFs usados são matematicamente válidos.

INSERT INTO clientes (id, nome, cpf_cnpj, email, telefone) VALUES
  (
    '22000000-0000-0000-0000-000000000001',
    'João Carlos Silva',
    '52998224725',                          -- CPF: 529.982.247-25
    'joao.silva@email.com',
    '(11) 91234-5678'
  ),
  (
    '22000000-0000-0000-0000-000000000002',
    'Maria Aparecida Santos',
    '71429964067',                          -- CPF: 714.299.640-67
    'maria.santos@email.com',
    '(11) 98765-4321'
  ),
  (
    '22000000-0000-0000-0000-000000000003',
    'Carlos Eduardo Oliveira',
    '32643057034',                          -- CPF: 326.430.570-34
    'carlos.oliveira@email.com',
    '(21) 99876-5432'
  ),
  (
    '22000000-0000-0000-0000-000000000004',
    'Auto Peças Rápido Ltda',
    '11222333000181',                       -- CNPJ: 11.222.333/0001-81
    'contato@autopecasrapido.com.br',
    '(11) 3456-7890'
  ),
  (
    '22000000-0000-0000-0000-000000000005',
    'Transportes Veloz S/A',
    '10621135000180',                       -- CNPJ: 10.621.135/0001-80
    'frota@transportesveloz.com.br',
    '(11) 4567-8901'
  )
ON CONFLICT (cpf_cnpj) DO NOTHING;


-- ════════════════════════════════════════════════════════════
-- 2. VEÍCULOS
-- ════════════════════════════════════════════════════════════

INSERT INTO veiculos (id, cliente_id, placa, marca, modelo, ano, cor) VALUES
  -- João Carlos Silva (2 veículos)
  ('33000000-0000-0000-0000-000000000001', '22000000-0000-0000-0000-000000000001', 'ABC1234',  'Volkswagen', 'Gol',     2018, 'Prata'),
  ('33000000-0000-0000-0000-000000000006', '22000000-0000-0000-0000-000000000001', 'QRS5E67',  'Volkswagen', 'T-Cross', 2023, 'Branco'),

  -- Maria Aparecida Santos (2 veículos)
  ('33000000-0000-0000-0000-000000000002', '22000000-0000-0000-0000-000000000002', 'DEF5678',  'Chevrolet',  'Onix',    2020, 'Vermelho'),
  ('33000000-0000-0000-0000-000000000007', '22000000-0000-0000-0000-000000000002', 'TUV6F78',  'Honda',      'Civic',   2021, 'Preto'),

  -- Carlos Eduardo Oliveira (2 veículos)
  ('33000000-0000-0000-0000-000000000003', '22000000-0000-0000-0000-000000000003', 'GHI9012',  'Toyota',     'Corolla', 2022, 'Cinza'),
  ('33000000-0000-0000-0000-000000000008', '22000000-0000-0000-0000-000000000003', 'WXY7G89',  'Renault',    'Kwid',    2021, 'Azul'),

  -- Auto Peças Rápido (frota — 1 veículo)
  ('33000000-0000-0000-0000-000000000004', '22000000-0000-0000-0000-000000000004', 'JKL3456',  'Fiat',       'Uno',     2015, 'Branco'),

  -- Transportes Veloz (frota — 1 veículo)
  ('33000000-0000-0000-0000-000000000005', '22000000-0000-0000-0000-000000000005', 'MNO7890',  'Ford',       'Ka',      2019, 'Preto')
ON CONFLICT (placa) DO NOTHING;


-- ════════════════════════════════════════════════════════════
-- 3. ORDENS DE SERVIÇO
-- ════════════════════════════════════════════════════════════
-- Cada OS cobre um status diferente para testar todos os fluxos.
-- Os timestamps refletem uma progressão realista de dias.
--
-- STATUS UUIDs (da migration 000002):
--   recebida             → 660e8400-e29b-41d4-a716-446655440001
--   em_diagnostico       → 660e8400-e29b-41d4-a716-446655440002
--   aguardando_aprovacao → 660e8400-e29b-41d4-a716-446655440003
--   em_execucao          → 660e8400-e29b-41d4-a716-446655440004
--   finalizada           → 660e8400-e29b-41d4-a716-446655440005
--   entregue             → 660e8400-e29b-41d4-a716-446655440006

INSERT INTO ordens_servico (
  id, numero, cliente_id, veiculo_id, usuario_responsavel_id, status_id,
  descricao, diagnostico,
  valor_total,
  aprovado_em, reprovado_em, iniciado_em, finalizado_em, entregue_em,
  criado_em, atualizado_em
) VALUES

  -- ── OS-2026-00001: recebida (aguardando início do diagnóstico) ───────────
  -- Endpoint útil: PATCH /work-orders/{id}/status → {"status":"em_diagnostico"}
  (
    '66000000-0000-0000-0000-000000000001',
    'OS-2026-00001',
    '22000000-0000-0000-0000-000000000001',  -- João Silva
    '33000000-0000-0000-0000-000000000001',  -- VW Gol 2018
    NULL,
    '660e8400-e29b-41d4-a716-446655440001',  -- recebida
    'Veículo com barulho estranho no motor e consumo de óleo elevado. Cliente relata fumaça azulada no escapamento.',
    NULL,
    240.60, NULL, NULL, NULL, NULL, NULL,
    '2026-04-28 09:00:00', '2026-04-28 09:00:00'
  ),

  -- ── OS-2026-00002: em_diagnostico ─────────────────────────────────────────
  -- Endpoint útil: PATCH /work-orders/{id}/status → {"status":"aguardando_aprovacao","diagnostico":"..."}
  (
    '66000000-0000-0000-0000-000000000002',
    'OS-2026-00002',
    '22000000-0000-0000-0000-000000000002',  -- Maria Santos
    '33000000-0000-0000-0000-000000000002',  -- Chevrolet Onix 2020
    '11000000-0000-0000-0000-000000000002',  -- João Pedro Mecânico
    '660e8400-e29b-41d4-a716-446655440002',  -- em_diagnostico
    'Pedal de freio esponjoso e veículo demora para parar. Cliente relata leve puxão para a esquerda ao frear.',
    NULL,
    150.00, NULL, NULL, NULL, NULL, NULL,
    '2026-04-25 10:00:00', '2026-04-25 11:30:00'
  ),

  -- ── OS-2026-00003: aguardando_aprovacao ────────────────────────────────────
  -- Endpoint útil:
  --   POST /work-orders/{id}/approve    → aprova e vai para em_execucao
  --   POST /work-orders/{id}/reject     → rejeita com motivo
  (
    '66000000-0000-0000-0000-000000000003',
    'OS-2026-00003',
    '22000000-0000-0000-0000-000000000003',  -- Carlos Oliveira
    '33000000-0000-0000-0000-000000000003',  -- Toyota Corolla 2022
    '11000000-0000-0000-0000-000000000002',  -- João Pedro Mecânico
    '660e8400-e29b-41d4-a716-446655440003',  -- aguardando_aprovacao
    'Veículo com solavanco ao acelerar e luz de falha acesa. Suspeita de correia dentada gasta.',
    'Correia dentada com desgaste acima do limite de 60.000 km. Tensor com folga excessiva. Recomendo substituição imediata para evitar dano ao motor.',
    750.00, NULL, NULL, NULL, NULL, NULL,
    '2026-04-20 14:00:00', '2026-04-21 10:00:00'
  ),

  -- ── OS-2026-00004: em_execucao ─────────────────────────────────────────────
  -- Endpoint útil: PATCH /work-orders/{id}/status → {"status":"finalizada"}
  (
    '66000000-0000-0000-0000-000000000004',
    'OS-2026-00004',
    '22000000-0000-0000-0000-000000000004',  -- Auto Peças Rápido
    '33000000-0000-0000-0000-000000000004',  -- Fiat Uno 2015
    '11000000-0000-0000-0000-000000000002',  -- João Pedro Mecânico
    '660e8400-e29b-41d4-a716-446655440004',  -- em_execucao
    'Revisão preventiva de rotina. Alinhamento e balanceamento solicitados pelo cliente antes de viagem longa.',
    'Alinhamento necessário (desvio de 2mm na roda direita). Balanceamento das 4 rodas. Sem outras irregularidades.',
    120.00,
    '2026-04-16 09:00:00', NULL, '2026-04-16 09:30:00', NULL, NULL,
    '2026-04-15 08:00:00', '2026-04-16 09:30:00'
  ),

  -- ── OS-2026-00005: finalizada (serviço concluído) ──────────────────────────
  -- Endpoint útil: PATCH /work-orders/{id}/status → {"status":"entregue"}
  (
    '66000000-0000-0000-0000-000000000005',
    'OS-2026-00005',
    '22000000-0000-0000-0000-000000000005',  -- Transportes Veloz
    '33000000-0000-0000-0000-000000000005',  -- Ford Ka 2019
    '11000000-0000-0000-0000-000000000002',  -- João Pedro Mecânico
    '660e8400-e29b-41d4-a716-446655440005',  -- finalizada
    'Motor falhando e dificuldade para dar partida a frio. Último serviço há 30.000 km.',
    'Velas de ignição com desgaste severo (eletrodo erodido). Substituição das 4 velas e limpeza do sistema de ignição.',
    250.00,
    '2026-04-11 08:00:00', NULL, '2026-04-11 08:30:00', '2026-04-11 10:00:00', NULL,
    '2026-04-10 11:00:00', '2026-04-11 10:00:00'
  ),

  -- ── OS-2026-00006: entregue ────────────────────────────────────────────────
  -- OS encerrada: útil para relatório de tempo médio
  (
    '66000000-0000-0000-0000-000000000006',
    'OS-2026-00006',
    '22000000-0000-0000-0000-000000000001',  -- João Silva
    '33000000-0000-0000-0000-000000000006',  -- VW T-Cross 2023
    '11000000-0000-0000-0000-000000000002',  -- João Pedro Mecânico
    '660e8400-e29b-41d4-a716-446655440006',  -- entregue
    'Check-up completo solicitado pelo cliente. Veículo novo, primeira revisão.',
    'Diagnóstico eletrônico sem falhas ativas. Todos os sistemas dentro do esperado. Nenhuma intervenção necessária.',
    120.00,
    '2026-04-06 09:00:00', NULL, '2026-04-06 09:30:00', '2026-04-06 10:30:00', '2026-04-07 11:00:00',
    '2026-04-05 09:00:00', '2026-04-07 11:00:00'
  ),

  -- ── OS-2026-00007: finalizada (orçamento rejeitado pelo cliente) ───────────
  -- reprovado_em preenchido; iniciado_em e finalizado_em nulos (não foi executado)
  (
    '66000000-0000-0000-0000-000000000007',
    'OS-2026-00007',
    '22000000-0000-0000-0000-000000000002',  -- Maria Santos
    '33000000-0000-0000-0000-000000000007',  -- Honda Civic 2021
    '11000000-0000-0000-0000-000000000002',  -- João Pedro Mecânico
    '660e8400-e29b-41d4-a716-446655440005',  -- finalizada
    'Suspensão batendo em irregularidades. Barulho ao passar em lombadas e buracos.',
    'Amortecedores dianteiros com vazamento de óleo. Necessária substituição do par. Estimativa: R$ 1.040,00.',
    1040.00,
    NULL, '2026-04-03 15:00:00', NULL, NULL, NULL,
    '2026-04-01 15:00:00', '2026-04-03 15:00:00'
  )
ON CONFLICT (numero) DO NOTHING;

-- Sincroniza a sequence de número da OS com a quantidade inserida
SELECT setval('os_numero_seq', 7);


-- ════════════════════════════════════════════════════════════
-- 4. ITENS DAS ORDENS DE SERVIÇO
-- ════════════════════════════════════════════════════════════

-- ── Serviços por OS ───────────────────────────────────────────────────────────

INSERT INTO itens_os_servicos (id, os_id, servico_id, quantidade, preco_unitario) VALUES
  -- OS-1: Troca de óleo e filtro
  ('77000000-0000-0000-0000-000000000001', '66000000-0000-0000-0000-000000000001', '44000000-0000-0000-0000-000000000001', 1,  80.00),

  -- OS-2: Revisão de freios dianteiros
  ('77000000-0000-0000-0000-000000000002', '66000000-0000-0000-0000-000000000002', '44000000-0000-0000-0000-000000000005', 1, 150.00),

  -- OS-3: Troca de correia dentada
  ('77000000-0000-0000-0000-000000000003', '66000000-0000-0000-0000-000000000003', '44000000-0000-0000-0000-000000000007', 1, 350.00),

  -- OS-4: Alinhamento e balanceamento
  ('77000000-0000-0000-0000-000000000004', '66000000-0000-0000-0000-000000000004', '44000000-0000-0000-0000-000000000004', 1, 120.00),

  -- OS-5: Troca de velas de ignição
  ('77000000-0000-0000-0000-000000000005', '66000000-0000-0000-0000-000000000005', '44000000-0000-0000-0000-000000000011', 1, 150.00),

  -- OS-6: Diagnóstico eletrônico
  ('77000000-0000-0000-0000-000000000006', '66000000-0000-0000-0000-000000000006', '44000000-0000-0000-0000-000000000008', 1, 120.00),

  -- OS-7: Troca de amortecedores (orçamento rejeitado)
  ('77000000-0000-0000-0000-000000000007', '66000000-0000-0000-0000-000000000007', '44000000-0000-0000-0000-000000000012', 1, 400.00)
ON CONFLICT (id) DO NOTHING;

-- ── Peças por OS ──────────────────────────────────────────────────────────────

INSERT INTO itens_os_pecas (id, os_id, peca_id, quantidade, preco_unitario) VALUES
  -- OS-1: Filtro de óleo (1x) + Óleo 5W30 (4L)
  ('88000000-0000-0000-0000-000000000001', '66000000-0000-0000-0000-000000000001', '55000000-0000-0000-0000-000000000003', 1,  45.00),
  ('88000000-0000-0000-0000-000000000002', '66000000-0000-0000-0000-000000000001', '55000000-0000-0000-0000-000000000001', 4,  28.90),

  -- OS-3: Correia dentada Gates (1x) + Tensor (1x)
  ('88000000-0000-0000-0000-000000000003', '66000000-0000-0000-0000-000000000003', '55000000-0000-0000-0000-000000000010', 1, 280.00),
  ('88000000-0000-0000-0000-000000000004', '66000000-0000-0000-0000-000000000003', '55000000-0000-0000-0000-000000000011', 1, 120.00),

  -- OS-5: Velas NGK (4x) — estoque já deduzido no INSERT de pecas
  ('88000000-0000-0000-0000-000000000005', '66000000-0000-0000-0000-000000000005', '55000000-0000-0000-0000-000000000012', 4,  25.00),

  -- OS-7: Amortecedor dianteiro Monroe (2x) — orçamento rejeitado, estoque NÃO deduzido
  ('88000000-0000-0000-0000-000000000006', '66000000-0000-0000-0000-000000000007', '55000000-0000-0000-0000-000000000014', 2, 320.00)
ON CONFLICT (id) DO NOTHING;


-- ════════════════════════════════════════════════════════════
-- 5. HISTÓRICO DE STATUS
-- ════════════════════════════════════════════════════════════
-- Registra a trilha de auditoria de cada transição de status.
-- status_anterior_id = NULL indica criação da OS.

INSERT INTO historicos_status (id, os_id, status_anterior_id, status_novo_id, alterado_por_usuario_id, observacao, alterado_em) VALUES

  -- ── OS-1: recebida (só criação) ───────────────────────────────────────────
  ('99000000-0000-0000-0000-000000000001',
    '66000000-0000-0000-0000-000000000001',
    NULL,
    '660e8400-e29b-41d4-a716-446655440001',  -- → recebida
    '11000000-0000-0000-0000-000000000003',  -- Ana (atendente abriu a OS)
    'OS aberta após recepção do veículo.',
    '2026-04-28 09:00:00'),

  -- ── OS-2: recebida → em_diagnostico ──────────────────────────────────────
  ('99000000-0000-0000-0000-000000000002',
    '66000000-0000-0000-0000-000000000002',
    NULL,
    '660e8400-e29b-41d4-a716-446655440001',
    '11000000-0000-0000-0000-000000000003',
    'OS aberta. Veículo entregue à mecânica.',
    '2026-04-25 10:00:00'),
  ('99000000-0000-0000-0000-000000000003',
    '66000000-0000-0000-0000-000000000002',
    '660e8400-e29b-41d4-a716-446655440001',  -- recebida
    '660e8400-e29b-41d4-a716-446655440002',  -- → em_diagnostico
    '11000000-0000-0000-0000-000000000002',  -- João Pedro iniciou diagnóstico
    'Início do diagnóstico do sistema de freios.',
    '2026-04-25 11:30:00'),

  -- ── OS-3: recebida → em_diagnostico → aguardando_aprovacao ───────────────
  ('99000000-0000-0000-0000-000000000004',
    '66000000-0000-0000-0000-000000000003',
    NULL,
    '660e8400-e29b-41d4-a716-446655440001',
    '11000000-0000-0000-0000-000000000003',
    'OS aberta. Veículo entregue à mecânica.',
    '2026-04-20 14:00:00'),
  ('99000000-0000-0000-0000-000000000005',
    '66000000-0000-0000-0000-000000000003',
    '660e8400-e29b-41d4-a716-446655440001',
    '660e8400-e29b-41d4-a716-446655440002',
    '11000000-0000-0000-0000-000000000002',
    'Início do diagnóstico. Suspeita de correia dentada.',
    '2026-04-20 15:00:00'),
  ('99000000-0000-0000-0000-000000000006',
    '66000000-0000-0000-0000-000000000003',
    '660e8400-e29b-41d4-a716-446655440002',
    '660e8400-e29b-41d4-a716-446655440003',
    '11000000-0000-0000-0000-000000000002',
    'Diagnóstico concluído. Orçamento enviado ao cliente via WhatsApp.',
    '2026-04-21 10:00:00'),

  -- ── OS-4: recebida → em_diagnostico → aguardando_aprovacao → em_execucao ─
  ('99000000-0000-0000-0000-000000000007',
    '66000000-0000-0000-0000-000000000004',
    NULL,
    '660e8400-e29b-41d4-a716-446655440001',
    '11000000-0000-0000-0000-000000000003',
    'OS aberta para revisão preventiva.',
    '2026-04-15 08:00:00'),
  ('99000000-0000-0000-0000-000000000008',
    '66000000-0000-0000-0000-000000000004',
    '660e8400-e29b-41d4-a716-446655440001',
    '660e8400-e29b-41d4-a716-446655440002',
    '11000000-0000-0000-0000-000000000002',
    'Veículo no elevador para inspeção.',
    '2026-04-15 09:00:00'),
  ('99000000-0000-0000-0000-000000000009',
    '66000000-0000-0000-0000-000000000004',
    '660e8400-e29b-41d4-a716-446655440002',
    '660e8400-e29b-41d4-a716-446655440003',
    '11000000-0000-0000-0000-000000000002',
    'Diagnóstico concluído. Orçamento aprovado verbalmente, aguardando confirmação formal.',
    '2026-04-15 10:00:00'),
  ('99000000-0000-0000-0000-000000000010',
    '66000000-0000-0000-0000-000000000004',
    '660e8400-e29b-41d4-a716-446655440003',
    '660e8400-e29b-41d4-a716-446655440004',
    '11000000-0000-0000-0000-000000000003',
    'Cliente aprovou o orçamento por WhatsApp. Serviço iniciado.',
    '2026-04-16 09:00:00'),

  -- ── OS-5: → finalizada (todas as 5 transições) ───────────────────────────
  ('99000000-0000-0000-0000-000000000011',
    '66000000-0000-0000-0000-000000000005',
    NULL,
    '660e8400-e29b-41d4-a716-446655440001',
    '11000000-0000-0000-0000-000000000003',
    'OS aberta pelo atendente.',
    '2026-04-10 11:00:00'),
  ('99000000-0000-0000-0000-000000000012',
    '66000000-0000-0000-0000-000000000005',
    '660e8400-e29b-41d4-a716-446655440001',
    '660e8400-e29b-41d4-a716-446655440002',
    '11000000-0000-0000-0000-000000000002',
    'Diagnóstico iniciado. Suspeita de velas desgastadas.',
    '2026-04-10 12:00:00'),
  ('99000000-0000-0000-0000-000000000013',
    '66000000-0000-0000-0000-000000000005',
    '660e8400-e29b-41d4-a716-446655440002',
    '660e8400-e29b-41d4-a716-446655440003',
    '11000000-0000-0000-0000-000000000002',
    'Diagnóstico concluído. Orçamento de R$ 250,00 enviado.',
    '2026-04-10 14:00:00'),
  ('99000000-0000-0000-0000-000000000014',
    '66000000-0000-0000-0000-000000000005',
    '660e8400-e29b-41d4-a716-446655440003',
    '660e8400-e29b-41d4-a716-446655440004',
    '11000000-0000-0000-0000-000000000003',
    'Aprovado pelo cliente. Serviço iniciado.',
    '2026-04-11 08:00:00'),
  ('99000000-0000-0000-0000-000000000015',
    '66000000-0000-0000-0000-000000000005',
    '660e8400-e29b-41d4-a716-446655440004',
    '660e8400-e29b-41d4-a716-446655440005',
    '11000000-0000-0000-0000-000000000002',
    'Velas substituídas. Motor em perfeito funcionamento. Serviço concluído.',
    '2026-04-11 10:00:00'),

  -- ── OS-6: → entregue (todas as 6 transições) ─────────────────────────────
  ('99000000-0000-0000-0000-000000000016',
    '66000000-0000-0000-0000-000000000006',
    NULL,
    '660e8400-e29b-41d4-a716-446655440001',
    '11000000-0000-0000-0000-000000000003',
    'OS aberta para check-up de carro novo.',
    '2026-04-05 09:00:00'),
  ('99000000-0000-0000-0000-000000000017',
    '66000000-0000-0000-0000-000000000006',
    '660e8400-e29b-41d4-a716-446655440001',
    '660e8400-e29b-41d4-a716-446655440002',
    '11000000-0000-0000-0000-000000000002',
    'Diagnóstico via OBD2 iniciado.',
    '2026-04-06 09:00:00'),
  ('99000000-0000-0000-0000-000000000018',
    '66000000-0000-0000-0000-000000000006',
    '660e8400-e29b-41d4-a716-446655440002',
    '660e8400-e29b-41d4-a716-446655440003',
    '11000000-0000-0000-0000-000000000002',
    'Diagnóstico concluído. Nenhuma falha encontrada. Orçamento: R$ 120,00 (apenas mão de obra).',
    '2026-04-06 09:20:00'),
  ('99000000-0000-0000-0000-000000000019',
    '66000000-0000-0000-0000-000000000006',
    '660e8400-e29b-41d4-a716-446655440003',
    '660e8400-e29b-41d4-a716-446655440004',
    '11000000-0000-0000-0000-000000000003',
    'Aprovado. Relatório de diagnóstico sendo gerado.',
    '2026-04-06 09:30:00'),
  ('99000000-0000-0000-0000-000000000020',
    '66000000-0000-0000-0000-000000000006',
    '660e8400-e29b-41d4-a716-446655440004',
    '660e8400-e29b-41d4-a716-446655440005',
    '11000000-0000-0000-0000-000000000002',
    'Relatório entregue ao cliente. Nenhuma intervenção necessária.',
    '2026-04-06 10:30:00'),
  ('99000000-0000-0000-0000-000000000021',
    '66000000-0000-0000-0000-000000000006',
    '660e8400-e29b-41d4-a716-446655440005',
    '660e8400-e29b-41d4-a716-446655440006',
    '11000000-0000-0000-0000-000000000003',
    'Veículo entregue ao cliente com relatório impresso.',
    '2026-04-07 11:00:00'),

  -- ── OS-7: recebida → em_diagnostico → aguardando_aprovacao → finalizada (rejeitada) ─
  ('99000000-0000-0000-0000-000000000022',
    '66000000-0000-0000-0000-000000000007',
    NULL,
    '660e8400-e29b-41d4-a716-446655440001',
    '11000000-0000-0000-0000-000000000003',
    'OS aberta. Veículo com problema na suspensão.',
    '2026-04-01 15:00:00'),
  ('99000000-0000-0000-0000-000000000023',
    '66000000-0000-0000-0000-000000000007',
    '660e8400-e29b-41d4-a716-446655440001',
    '660e8400-e29b-41d4-a716-446655440002',
    '11000000-0000-0000-0000-000000000002',
    'Iniciado diagnóstico da suspensão.',
    '2026-04-02 09:00:00'),
  ('99000000-0000-0000-0000-000000000024',
    '66000000-0000-0000-0000-000000000007',
    '660e8400-e29b-41d4-a716-446655440002',
    '660e8400-e29b-41d4-a716-446655440003',
    '11000000-0000-0000-0000-000000000002',
    'Amortecedores dianteiros com vazamento. Orçamento de R$ 1.040,00 enviado ao cliente.',
    '2026-04-02 11:00:00'),
  ('99000000-0000-0000-0000-000000000025',
    '66000000-0000-0000-0000-000000000007',
    '660e8400-e29b-41d4-a716-446655440003',
    '660e8400-e29b-41d4-a716-446655440005',
    '11000000-0000-0000-0000-000000000003',
    'Cliente rejeitou o orçamento. Motivo: valor acima do esperado. Veículo liberado.',
    '2026-04-03 15:00:00')
ON CONFLICT (id) DO NOTHING;
