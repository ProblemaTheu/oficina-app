CREATE TABLE "papeis_usuario" (
   "id" uuid PRIMARY KEY NOT NULL,
   "nome_papel" varchar(50) UNIQUE NOT NULL,
   "descricao" text,
   "criado_em" timestamp NOT NULL DEFAULT (now()),
   "atualizado_em" timestamp NOT NULL DEFAULT (now())
 );

CREATE TABLE "status_ordens" (
   "id" uuid PRIMARY KEY NOT NULL,
   "nome_status" varchar(50) UNIQUE NOT NULL,
   "descricao" text,
   "criado_em" timestamp NOT NULL DEFAULT (now()),
   "atualizado_em" timestamp NOT NULL DEFAULT (now())
 );

CREATE TABLE "clientes" (
   "id" uuid PRIMARY KEY NOT NULL,
   "nome" varchar(255) NOT NULL,
   "cpf_cnpj" varchar(20) UNIQUE NOT NULL,
   "email" varchar(255) UNIQUE,
   "telefone" varchar(20),
   "criado_em" timestamp NOT NULL DEFAULT (now()),
   "atualizado_em" timestamp NOT NULL DEFAULT (now())
 );

CREATE TABLE "usuarios" (
   "id" uuid PRIMARY KEY NOT NULL,
   "nome" varchar(255) NOT NULL,
   "email" varchar(255) UNIQUE NOT NULL,
   "senha_hash" varchar(255) NOT NULL,
   "papel_id" uuid NOT NULL,
   "criado_em" timestamp NOT NULL DEFAULT (now()),
   "atualizado_em" timestamp NOT NULL DEFAULT (now())
 );

CREATE TABLE "veiculos" (
   "id" uuid PRIMARY KEY NOT NULL,
   "cliente_id" uuid NOT NULL,
   "placa" varchar(20) UNIQUE NOT NULL,
   "marca" varchar(100) NOT NULL,
   "modelo" varchar(100) NOT NULL,
   "ano" int NOT NULL,
   "cor" varchar(50),
   "criado_em" timestamp NOT NULL DEFAULT (now()),
   "atualizado_em" timestamp NOT NULL DEFAULT (now())
 );

CREATE TABLE "ordens_servico" (
   "id" uuid PRIMARY KEY NOT NULL,
   "cliente_id" uuid NOT NULL,
   "veiculo_id" uuid NOT NULL,
   "usuario_responsavel_id" uuid,
   "status_id" uuid NOT NULL,
   "valor_total" decimal(10,2) NOT NULL DEFAULT 0,
   "criado_em" timestamp NOT NULL DEFAULT (now()),
   "atualizado_em" timestamp NOT NULL DEFAULT (now())
 );

CREATE TABLE "servicos" (
   "id" uuid PRIMARY KEY NOT NULL,
   "nome" varchar(255) UNIQUE NOT NULL,
   "descricao" text,
   "preco_base" decimal(10,2) NOT NULL DEFAULT 0,
   "tempo_minutos" int NOT NULL DEFAULT 0,
   "criado_em" timestamp NOT NULL DEFAULT (now()),
   "atualizado_em" timestamp NOT NULL DEFAULT (now())
 );

CREATE TABLE "pecas" (
   "id" uuid PRIMARY KEY NOT NULL,
   "nome" varchar(255) NOT NULL,
   "codigo" varchar(100) UNIQUE NOT NULL,
   "preco" decimal(10,2) NOT NULL DEFAULT 0,
   "estoque_atual" int NOT NULL DEFAULT 0,
   "estoque_minimo" int NOT NULL DEFAULT 0,
   "criado_em" timestamp NOT NULL DEFAULT (now()),
   "atualizado_em" timestamp NOT NULL DEFAULT (now())
 );

CREATE TABLE "itens_os_servicos" (
   "id" uuid PRIMARY KEY NOT NULL,
   "os_id" uuid NOT NULL,
   "servico_id" uuid NOT NULL,
   "quantidade" int NOT NULL DEFAULT 1,
   "preco_unitario" decimal(10,2) NOT NULL DEFAULT 0,
   "criado_em" timestamp NOT NULL DEFAULT (now()),
   "atualizado_em" timestamp NOT NULL DEFAULT (now())
 );

CREATE TABLE "itens_os_pecas" (
   "id" uuid PRIMARY KEY NOT NULL,
   "os_id" uuid NOT NULL,
   "peca_id" uuid NOT NULL,
   "quantidade" int NOT NULL DEFAULT 1,
   "preco_unitario" decimal(10,2) NOT NULL DEFAULT 0,
   "criado_em" timestamp NOT NULL DEFAULT (now()),
   "atualizado_em" timestamp NOT NULL DEFAULT (now())
 );

CREATE TABLE "historicos_status" (
   "id" uuid PRIMARY KEY NOT NULL,
   "os_id" uuid NOT NULL,
   "status_anterior_id" uuid,
   "status_novo_id" uuid NOT NULL,
   "alterado_em" timestamp NOT NULL DEFAULT (now()),
   "alterado_por_usuario_id" uuid,
   "criado_em" timestamp NOT NULL DEFAULT (now()),
   "atualizado_em" timestamp NOT NULL DEFAULT (now())
 );

COMMENT ON TABLE "papeis_usuario" IS 'Define os diferentes papéis que um usuário pode ter no sistema.';
COMMENT ON COLUMN "papeis_usuario"."nome_papel" IS 'Nome único do papel do usuário (ex: administrador, mecanico, cliente).';
COMMENT ON COLUMN "papeis_usuario"."descricao" IS 'Descrição detalhada do papel do usuário.';
COMMENT ON COLUMN "papeis_usuario"."criado_em" IS 'Data e hora de criação do registro.';
COMMENT ON COLUMN "papeis_usuario"."atualizado_em" IS 'Data e hora da última atualização do registro.';
COMMENT ON TABLE "status_ordens" IS 'Define os possíveis status para as ordens de serviço.';
COMMENT ON COLUMN "status_ordens"."nome_status" IS 'Nome único do status da ordem de serviço (ex: pendente, em_progresso, concluida).';
COMMENT ON COLUMN "status_ordens"."descricao" IS 'Descrição detalhada do status da ordem de serviço.';
COMMENT ON COLUMN "status_ordens"."criado_em" IS 'Data e hora de criação do registro.';
COMMENT ON COLUMN "status_ordens"."atualizado_em" IS 'Data e hora da última atualização do registro.';
COMMENT ON TABLE "clientes" IS 'Armazena informações dos clientes.';
COMMENT ON COLUMN "clientes"."nome" IS 'Nome completo do cliente ou razão social.';
COMMENT ON COLUMN "clientes"."cpf_cnpj" IS 'CPF ou CNPJ do cliente.';
COMMENT ON COLUMN "clientes"."email" IS 'Endereço de e-mail do cliente.';
COMMENT ON COLUMN "clientes"."telefone" IS 'Número de telefone do cliente.';
COMMENT ON COLUMN "clientes"."criado_em" IS 'Data e hora de criação do registro.';
COMMENT ON COLUMN "clientes"."atualizado_em" IS 'Data e hora da última atualização do registro.';
COMMENT ON TABLE "usuarios" IS 'Armazena informações dos usuários do sistema.';
COMMENT ON COLUMN "usuarios"."nome" IS 'Nome completo do usuário.';
COMMENT ON COLUMN "usuarios"."email" IS 'Endereço de e-mail do usuário, usado para login.';
COMMENT ON COLUMN "usuarios"."senha_hash" IS 'Hash da senha do usuário.';
COMMENT ON COLUMN "usuarios"."papel_id" IS 'ID do papel do usuário no sistema.';
COMMENT ON COLUMN "usuarios"."criado_em" IS 'Data e hora de criação do registro.';
COMMENT ON COLUMN "usuarios"."atualizado_em" IS 'Data e hora da última atualização do registro.';
COMMENT ON TABLE "veiculos" IS 'Armazena informações dos veículos dos clientes.';
COMMENT ON COLUMN "veiculos"."cliente_id" IS 'ID do cliente proprietário do veículo.';
COMMENT ON COLUMN "veiculos"."placa" IS 'Placa do veículo.';
COMMENT ON COLUMN "veiculos"."marca" IS 'Marca do veículo.';
COMMENT ON COLUMN "veiculos"."modelo" IS 'Modelo do veículo.';
COMMENT ON COLUMN "veiculos"."ano" IS 'Ano de fabricação do veículo.';
COMMENT ON COLUMN "veiculos"."cor" IS 'Cor do veículo.';
COMMENT ON COLUMN "veiculos"."criado_em" IS 'Data e hora de criação do registro.';
COMMENT ON COLUMN "veiculos"."atualizado_em" IS 'Data e hora da última atualização do registro.';
COMMENT ON TABLE "ordens_servico" IS 'Registra as ordens de serviço.';
COMMENT ON COLUMN "ordens_servico"."cliente_id" IS 'ID do cliente associado à ordem de serviço.';
COMMENT ON COLUMN "ordens_servico"."veiculo_id" IS 'ID do veículo associado à ordem de serviço.';
COMMENT ON COLUMN "ordens_servico"."usuario_responsavel_id" IS 'ID do usuário (mecânico) responsável pela ordem de serviço.';
COMMENT ON COLUMN "ordens_servico"."status_id" IS 'ID do status atual da ordem de serviço.';
COMMENT ON COLUMN "ordens_servico"."valor_total" IS 'Valor total da ordem de serviço.';
COMMENT ON COLUMN "ordens_servico"."criado_em" IS 'Data e hora de criação da ordem de serviço.';
COMMENT ON COLUMN "ordens_servico"."atualizado_em" IS 'Data e hora da última atualização da ordem de serviço.';
COMMENT ON TABLE "servicos" IS 'Define os tipos de serviços oferecidos.';
COMMENT ON COLUMN "servicos"."nome" IS 'Nome do serviço.';
COMMENT ON COLUMN "servicos"."descricao" IS 'Descrição detalhada do serviço.';
COMMENT ON COLUMN "servicos"."preco_base" IS 'Preço base do serviço.';
COMMENT ON COLUMN "servicos"."tempo_minutos" IS 'Tempo estimado em minutos para a execução do serviço.';
COMMENT ON COLUMN "servicos"."criado_em" IS 'Data e hora de criação do registro.';
COMMENT ON COLUMN "servicos"."atualizado_em" IS 'Data e hora da última atualização do registro.';
COMMENT ON TABLE "pecas" IS 'Gerencia o estoque de peças.';
COMMENT ON COLUMN "pecas"."nome" IS 'Nome da peça.';
COMMENT ON COLUMN "pecas"."codigo" IS 'Código de identificação da peça.';
COMMENT ON COLUMN "pecas"."preco" IS 'Preço de venda da peça.';
COMMENT ON COLUMN "pecas"."estoque_atual" IS 'Quantidade atual da peça em estoque.';
COMMENT ON COLUMN "pecas"."estoque_minimo" IS 'Quantidade mínima da peça em estoque para alerta.';
COMMENT ON COLUMN "pecas"."criado_em" IS 'Data e hora de criação do registro.';
COMMENT ON COLUMN "pecas"."atualizado_em" IS 'Data e hora da última atualização do registro.';
COMMENT ON TABLE "itens_os_servicos" IS 'Detalhes dos serviços incluídos em uma ordem de serviço.';
COMMENT ON COLUMN "itens_os_servicos"."os_id" IS 'ID da ordem de serviço à qual o serviço pertence.';
COMMENT ON COLUMN "itens_os_servicos"."servico_id" IS 'ID do serviço.';
COMMENT ON COLUMN "itens_os_servicos"."quantidade" IS 'Quantidade do serviço.';
COMMENT ON COLUMN "itens_os_servicos"."preco_unitario" IS 'Preço unitário do serviço no momento da inclusão.';
COMMENT ON COLUMN "itens_os_servicos"."criado_em" IS 'Data e hora de criação do registro.';
COMMENT ON COLUMN "itens_os_servicos"."atualizado_em" IS 'Data e hora da última atualização do registro.';
COMMENT ON TABLE "itens_os_pecas" IS 'Detalhes das peças incluídas em uma ordem de serviço.';
COMMENT ON COLUMN "itens_os_pecas"."os_id" IS 'ID da ordem de serviço à qual a peça pertence.';
COMMENT ON COLUMN "itens_os_pecas"."peca_id" IS 'ID da peça.';
COMMENT ON COLUMN "itens_os_pecas"."quantidade" IS 'Quantidade da peça.';
COMMENT ON COLUMN "itens_os_pecas"."preco_unitario" IS 'Preço unitário da peça no momento da inclusão.';
COMMENT ON COLUMN "itens_os_pecas"."criado_em" IS 'Data e hora da última atualização do registro.';
COMMENT ON COLUMN "itens_os_pecas"."atualizado_em" IS 'Data e hora da última atualização do registro.';
COMMENT ON TABLE "historicos_status" IS 'Registra o histórico de mudanças de status das ordens de serviço.';
COMMENT ON COLUMN "historicos_status"."os_id" IS 'ID da ordem de serviço.';
COMMENT ON COLUMN "historicos_status"."status_anterior_id" IS 'ID do status anterior da ordem de serviço.';
COMMENT ON COLUMN "historicos_status"."status_novo_id" IS 'ID do novo status da ordem de serviço.';
COMMENT ON COLUMN "historicos_status"."alterado_em" IS 'Data e hora da alteração do status.';
COMMENT ON COLUMN "historicos_status"."alterado_por_usuario_id" IS 'ID do usuário que realizou a alteração do status.';
COMMENT ON COLUMN "historicos_status"."criado_em" IS 'Data e hora de criação do registro.';
COMMENT ON COLUMN "historicos_status"."atualizado_em" IS 'Data e hora da última atualização do registro.';

ALTER TABLE "usuarios" ADD FOREIGN KEY ("papel_id") REFERENCES "papeis_usuario" ("id");
ALTER TABLE "veiculos" ADD FOREIGN KEY ("cliente_id") REFERENCES "clientes" ("id");
ALTER TABLE "ordens_servico" ADD FOREIGN KEY ("cliente_id") REFERENCES "clientes" ("id");
ALTER TABLE "ordens_servico" ADD FOREIGN KEY ("veiculo_id") REFERENCES "veiculos" ("id");
ALTER TABLE "ordens_servico" ADD FOREIGN KEY ("usuario_responsavel_id") REFERENCES "usuarios" ("id");
ALTER TABLE "ordens_servico" ADD FOREIGN KEY ("status_id") REFERENCES "status_ordens" ("id");
ALTER TABLE "itens_os_servicos" ADD FOREIGN KEY ("os_id") REFERENCES "ordens_servico" ("id");
ALTER TABLE "itens_os_servicos" ADD FOREIGN KEY ("servico_id") REFERENCES "servicos" ("id");
ALTER TABLE "itens_os_pecas" ADD FOREIGN KEY ("os_id") REFERENCES "ordens_servico" ("id");
ALTER TABLE "itens_os_pecas" ADD FOREIGN KEY ("peca_id") REFERENCES "pecas" ("id");
ALTER TABLE "historicos_status" ADD FOREIGN KEY ("os_id") REFERENCES "ordens_servico" ("id");
ALTER TABLE "historicos_status" ADD FOREIGN KEY ("status_anterior_id") REFERENCES "status_ordens" ("id");
ALTER TABLE "historicos_status" ADD FOREIGN KEY ("status_novo_id") REFERENCES "status_ordens" ("id");
ALTER TABLE "historicos_status" ADD FOREIGN KEY ("alterado_por_usuario_id") REFERENCES "usuarios" ("id");
