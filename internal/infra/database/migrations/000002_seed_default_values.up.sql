INSERT INTO "papeis_usuario" ("id", "nome_papel", "descricao") VALUES
  ('550e8400-e29b-41d4-a716-446655440001', 'administrador', 'Acesso total ao sistema, gerenciamento de usuários e configurações.'),
  ('550e8400-e29b-41d4-a716-446655440002', 'mecanico',      'Responsável pela execução dos serviços nas ordens de serviço.'),
  ('550e8400-e29b-41d4-a716-446655440003', 'atendente',     'Responsável pelo atendimento ao cliente e abertura de ordens de serviço.');

INSERT INTO "status_ordens" ("id", "nome_status", "descricao") VALUES
  ('660e8400-e29b-41d4-a716-446655440001', 'recebida',              'OS criada, veículo recebido na oficina. Aguardando início do diagnóstico.'),
  ('660e8400-e29b-41d4-a716-446655440002', 'em_diagnostico',        'Mecânico está realizando o diagnóstico do veículo.'),
  ('660e8400-e29b-41d4-a716-446655440003', 'aguardando_aprovacao',  'Diagnóstico concluído. Orçamento enviado ao cliente, aguardando aprovação.'),
  ('660e8400-e29b-41d4-a716-446655440004', 'em_execucao',           'Cliente aprovou o orçamento. Serviço em execução.'),
  ('660e8400-e29b-41d4-a716-446655440005', 'finalizada',            'Serviço concluído ou orçamento rejeitado pelo cliente.'),
  ('660e8400-e29b-41d4-a716-446655440006', 'entregue',              'Veículo entregue ao cliente.');
