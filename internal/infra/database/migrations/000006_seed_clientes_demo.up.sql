-- Clientes de demonstração (F3-5.3).
--
-- Os dois CPFs passam na validação de dígitos verificadores — CPF inventado
-- é rejeitado pela Lambda antes de chegar ao banco, e o fluxo do vídeo
-- quebraria ao vivo. Conferidos com o algoritmo do módulo 11.
--
-- O cliente INATIVO existe para demonstrar o requisito "consultar a
-- existência E o status do cliente": sem ele não há como mostrar o 403.
--
-- IDs fixos de propósito: a migration é idempotente e os testes e o roteiro
-- do vídeo podem referenciá-los.

INSERT INTO "clientes" ("id", "nome", "cpf_cnpj", "email", "telefone", "status") VALUES
  ('770e8400-e29b-41d4-a716-446655440001', 'Maria Silva',    '529.982.247-25', 'maria@exemplo.com',  '11999990001', 'ativo'),
  ('770e8400-e29b-41d4-a716-446655440002', 'Carlos Pereira', '111.444.777-35', 'carlos@exemplo.com', '11999990002', 'inativo')
ON CONFLICT ("cpf_cnpj") DO NOTHING;

-- Um veículo por cliente: sem veículo não há como abrir OS, e a demonstração
-- do dia 3 termina em uma OS criada de verdade.
INSERT INTO "veiculos" ("id", "cliente_id", "placa", "marca", "modelo", "ano", "cor") VALUES
  ('880e8400-e29b-41d4-a716-446655440001', '770e8400-e29b-41d4-a716-446655440001', 'RRK1A23', 'Volkswagen', 'Gol',   2019, 'prata'),
  ('880e8400-e29b-41d4-a716-446655440002', '770e8400-e29b-41d4-a716-446655440002', 'RRK2B34', 'Fiat',       'Argo',  2021, 'branco')
ON CONFLICT ("placa") DO NOTHING;
