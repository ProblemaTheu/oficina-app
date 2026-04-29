INSERT INTO "papeis_usuario" ("id", "nome_papel", "descricao") VALUES
  ('550e8400-e29b-41d4-a716-446655440001', 'administrador', 'Acesso total ao sistema, gerenciamento de usuários e configurações.'),
  ('550e8400-e29b-41d4-a716-446655440002', 'mecanico',      'Responsável pela execução dos serviços nas ordens de serviço.'),
  ('550e8400-e29b-41d4-a716-446655440003', 'atendente',     'Responsável pelo atendimento ao cliente e abertura de ordens de serviço.');

INSERT INTO "status_ordens" ("id", "nome_status", "descricao") VALUES
  ('660e8400-e29b-41d4-a716-446655440001', 'pendente',          'Ordem de serviço criada, aguardando início.'),
  ('660e8400-e29b-41d4-a716-446655440002', 'em_andamento',      'Serviço em execução pelo mecânico responsável.'),
  ('660e8400-e29b-41d4-a716-446655440003', 'aguardando_pecas',  'Serviço pausado aguardando chegada de peças.'),
  ('660e8400-e29b-41d4-a716-446655440004', 'concluida',         'Todos os serviços foram finalizados.'),
  ('660e8400-e29b-41d4-a716-446655440005', 'cancelada',         'Ordem de serviço cancelada.'),
  ('660e8400-e29b-41d4-a716-446655440006', 'entregue',          'Veículo entregue ao cliente.');
