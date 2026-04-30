-- ============================================================
-- Migration 000004: Catálogo inicial da oficina mecânica
-- ============================================================
-- Popula os dados operacionais fixos que a oficina não precisa
-- cadastrar manualmente a cada nova instalação:
--   • Usuários de sistema (admin, mecânico, atendente)
--   • Catálogo de serviços
--   • Catálogo de peças com estoque inicial
--
-- Todos os INSERTs usam ON CONFLICT DO NOTHING, portanto é seguro
-- re-executar sem duplicar dados.
--
-- CREDENCIAIS:
--   admin@oficina.com  → Admin@123
--   joao@oficina.com   → Mecanico@123
--   ana@oficina.com    → Atende@123
-- ============================================================

-- ── Usuários ──────────────────────────────────────────────────────────────────

INSERT INTO usuarios (id, nome, email, senha_hash, papel_id) VALUES
  (
    '11000000-0000-0000-0000-000000000001',
    'Admin Sistema',
    'admin@oficina.com',
    '$2a$10$Wfldd9ninTKTZlScHzcISeeXu1S/CyVY9PbAruAB7bgIbbeAfxkgS',
    '550e8400-e29b-41d4-a716-446655440001'   -- administrador
  ),
  (
    '11000000-0000-0000-0000-000000000002',
    'João Pedro Mecânico',
    'joao@oficina.com',
    '$2a$10$YTO2a8pW6c5BNPpivFVn9OXCCyC.oJZrBjHk3IUztKLZoKJh7CDOe',
    '550e8400-e29b-41d4-a716-446655440002'   -- mecanico
  ),
  (
    '11000000-0000-0000-0000-000000000003',
    'Ana Lima Atendente',
    'ana@oficina.com',
    '$2a$10$.slUcKhDL7/0s16bLJUdl.PDO1YvK4MJKK68jP01JS2eOsBoulLwu',
    '550e8400-e29b-41d4-a716-446655440003'   -- atendente
  )
ON CONFLICT (email) DO NOTHING;

-- ── Serviços ──────────────────────────────────────────────────────────────────

INSERT INTO servicos (id, nome, descricao, preco_base, tempo_minutos) VALUES
  ('44000000-0000-0000-0000-000000000001', 'Troca de Óleo e Filtro',          'Substituição do óleo do motor e filtro de óleo. Inclui verificação do nível de todos os fluidos.',                             80.00,  45),
  ('44000000-0000-0000-0000-000000000002', 'Alinhamento de Direção',          'Correção do alinhamento das rodas para garantir dirigibilidade e evitar desgaste irregular dos pneus.',                        60.00,  30),
  ('44000000-0000-0000-0000-000000000003', 'Balanceamento de Rodas',          'Balanceamento das 4 rodas para eliminar vibrações e desgaste irregular de pneus.',                                              80.00,  60),
  ('44000000-0000-0000-0000-000000000004', 'Alinhamento e Balanceamento',     'Pacote completo: alinhamento de direção + balanceamento das 4 rodas.',                                                          120.00,  90),
  ('44000000-0000-0000-0000-000000000005', 'Revisão de Freios Dianteiros',    'Inspeção e substituição de pastilhas e discos dianteiros conforme necessidade. Inclui limpeza dos pinos de guia.',             150.00,  90),
  ('44000000-0000-0000-0000-000000000006', 'Revisão de Freios Traseiros',     'Inspeção e substituição de pastilhas/lonas e discos/tambores traseiros conforme necessidade.',                                  120.00,  60),
  ('44000000-0000-0000-0000-000000000007', 'Troca de Correia Dentada',        'Substituição da correia dentada, tensor e roldana. Recomendado a cada 60.000 km ou 4 anos.',                                    350.00, 180),
  ('44000000-0000-0000-0000-000000000008', 'Diagnóstico Eletrônico',          'Leitura de falhas via scanner automotivo OBD2. Relatório completo dos sistemas eletrônicos.',                                   120.00,  60),
  ('44000000-0000-0000-0000-000000000009', 'Higienização de Ar-Condicionado', 'Limpeza do evaporador, troca do filtro de cabine e aplicação de bactericida/fungicida.',                                       180.00,  90),
  ('44000000-0000-0000-0000-000000000010', 'Troca de Fluido de Freio',        'Substituição completa do fluido de freio DOT 4. Recomendado a cada 2 anos ou 40.000 km.',                                        80.00,  30),
  ('44000000-0000-0000-0000-000000000011', 'Troca de Velas de Ignição',       'Substituição das velas de ignição. Inclui verificação do sistema de ignição e ajuste de gap.',                                  150.00,  60),
  ('44000000-0000-0000-0000-000000000012', 'Troca de Amortecedores (par)',    'Substituição do par de amortecedores (dianteiro ou traseiro). Inclui alinhamento após o serviço.',                              400.00, 120),
  ('44000000-0000-0000-0000-000000000013', 'Revisão dos 10.000 km',           'Troca de óleo + filtros (óleo, ar e cabine). Verificação geral e check-up completo.',                                           300.00, 180),
  ('44000000-0000-0000-0000-000000000014', 'Revisão dos 50.000 km',           'Revisão completa: óleo, filtros, correia dentada, fluido de freio, velas e check-up eletrônico.',                               600.00, 240),
  ('44000000-0000-0000-0000-000000000015', 'Troca de Bateria',                'Substituição da bateria. Inclui teste de carga da bateria antiga e verificação do alternador.',                                   50.00,  20)
ON CONFLICT (nome) DO NOTHING;

-- ── Peças ─────────────────────────────────────────────────────────────────────
-- estoque_atual  = quantidade inicial em prateleira
-- estoque_minimo = ponto de reposição (alerta de estoque baixo)

INSERT INTO pecas (id, nome, codigo, preco, estoque_atual, estoque_minimo) VALUES
  ('55000000-0000-0000-0000-000000000001', 'Óleo Motor 5W30 Sintético 1L',         'OIL-5W30-1L',    28.90,  48, 20),
  ('55000000-0000-0000-0000-000000000002', 'Óleo Motor 10W40 Semissintético 1L',   'OIL-10W40-1L',   22.50,  60, 20),
  ('55000000-0000-0000-0000-000000000003', 'Filtro de Óleo Bosch',                 'FIL-OLEO-BSH',   45.00,  30, 10),
  ('55000000-0000-0000-0000-000000000004', 'Filtro de Ar Universal',               'FIL-AR-UNI',     38.00,  25,  8),
  ('55000000-0000-0000-0000-000000000005', 'Filtro de Combustível Flex',           'FIL-COMB-FLX',   52.00,  20,  5),
  ('55000000-0000-0000-0000-000000000006', 'Filtro de Cabine (Ar-Condicionado)',   'FIL-CAB-AC',     65.00,  22,  8),
  ('55000000-0000-0000-0000-000000000007', 'Pastilha de Freio Dianteira (jogo)',   'PAST-FRDN-JG',   89.90,  18,  6),
  ('55000000-0000-0000-0000-000000000008', 'Pastilha de Freio Traseira (jogo)',    'PAST-FRTRS-JG',  79.90,  16,  6),
  ('55000000-0000-0000-0000-000000000009', 'Disco de Freio Dianteiro',             'DISC-FRDN-UN',  145.00,  12,  4),
  ('55000000-0000-0000-0000-000000000010', 'Correia Dentada Gates',                'COR-DENT-GT',   280.00,   8,  3),
  ('55000000-0000-0000-0000-000000000011', 'Tensor da Correia Dentada',            'TENS-COR-DEN',  120.00,  10,  3),
  ('55000000-0000-0000-0000-000000000012', 'Vela de Ignição NGK (unidade)',        'VELA-NGK-UN',    25.00,  50, 16),
  ('55000000-0000-0000-0000-000000000013', 'Fluido de Freio DOT 4 (500ml)',        'FLUID-DOT4',     32.00,  20,  6),
  ('55000000-0000-0000-0000-000000000014', 'Amortecedor Dianteiro Monroe',         'AMOR-DN-MNR',   320.00,   6,  2),
  ('55000000-0000-0000-0000-000000000015', 'Amortecedor Traseiro Monroe',          'AMOR-TRS-MNR',  280.00,   6,  2),
  ('55000000-0000-0000-0000-000000000016', 'Rolamento de Roda Dianteiro',          'ROLM-ROD-DN',   185.00,   8,  2),
  ('55000000-0000-0000-0000-000000000017', 'Correia do Alternador',                'COR-ALT-UN',     55.00,  14,  4),
  ('55000000-0000-0000-0000-000000000018', 'Bateria 60Ah Heliar',                  'BAT-60AH-HLR',  420.00,   5,  2),
  ('55000000-0000-0000-0000-000000000019', 'Líquido de Arrefecimento (1L)',        'LIQ-ARF-1L',     18.50,  30, 10),
  ('55000000-0000-0000-0000-000000000020', 'Palheta Limpador de Para-brisa (par)', 'PAL-LIMP-PAR',   48.00,  15,  5)
ON CONFLICT (codigo) DO NOTHING;
