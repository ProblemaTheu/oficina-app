# Objetivos do Sistema — Tech Challenge Fase 1

## Problema que o sistema resolve

Oficinas mecânicas gerenciam hoje o fluxo de atendimento de forma manual ou em planilhas: o atendente anota o problema, passa para o mecânico, o mecânico faz o diagnóstico em papel, liga para o cliente para aprovação verbal, e no final há pouca rastreabilidade sobre o que foi feito, quanto tempo levou e quais peças foram usadas.

Este sistema digitaliza esse fluxo de ponta a ponta — desde a identificação do cliente pelo CPF até a entrega do veículo — com histórico completo de status, controle automático de estoque e relatórios de tempo médio de atendimento.

---

## Escopo do Desafio Técnico

Desenvolvimento de uma **API REST em Go** que implemente o back-end de um sistema de gestão de oficina mecânica, atendendo aos requisitos de:

- Modelagem de domínio com DDD (agregados, value objects, linguagem ubíqua)
- Arquitetura limpa com separação de camadas
- Persistência relacional com PostgreSQL
- Autenticação stateless com JWT
- Containerização com Docker
- Testes automatizados
- Análise de segurança via pipeline CI/CD

---

## Requisitos Funcionais Atendidos

### Autenticação e Autorização
- [x] Registro de usuários com papel (mecânico, atendente, administrador)
- [x] Login com e-mail e senha, retornando JWT válido por 8 horas
- [x] Todas as rotas protegidas por JWT (exceto login, registro e status público de OS)

### Clientes
- [x] Cadastrar cliente com CPF ou CNPJ (validação completa com dígitos verificadores)
- [x] Buscar cliente por CPF/CNPJ (`GET /v1/clients?documento=`)
- [x] Listar, buscar por ID, atualizar e remover clientes

### Veículos
- [x] Cadastrar veículo vinculado a um cliente, com validação de placa (formato antigo e Mercosul)
- [x] Listar veículos por cliente
- [x] Listar, buscar por ID, atualizar e remover veículos

### Catálogo de Serviços
- [x] Cadastrar serviços com preço base e tempo estimado
- [x] Listar, buscar, atualizar e remover serviços

### Controle de Estoque de Peças
- [x] Cadastrar peças com código interno, preço, estoque atual e estoque mínimo
- [x] Ajustar estoque manualmente (entrada, saída, ajuste)
- [x] Dedução automática de estoque ao iniciar execução de uma OS
- [x] Listar, buscar, atualizar e remover peças

### Ordens de Serviço
- [x] Abrir OS com itens (serviços e/ou peças), gerando número único automático
- [x] Máquina de estados com transições validadas pelo domínio
- [x] Registrar diagnóstico na transição para `aguardando_aprovacao`
- [x] Aprovação e reprovação de orçamento pelo cliente
- [x] Histórico imutável de todas as transições de status
- [x] Consulta pública de status de uma OS (sem autenticação)
- [x] Listar e buscar OS com detalhes completos (cliente, veículo, itens, histórico)

### Relatórios
- [x] Relatório de tempo médio de atendimento por período

### Saúde da aplicação
- [x] `GET /health/ready` — liveness + readiness com verificação do banco

---

## Requisitos Não-Funcionais

### Segurança
- Senhas armazenadas exclusivamente como hash bcrypt (custo 10)
- JWT assinado com HS256; segredo configurável via variável de ambiente
- Validação de CPF/CNPJ e placa no nível de value object (não apenas regex)
- Pipeline de segurança automatizado: `govulncheck`, `gosec`, `Trivy` e SonarCloud a cada push
- SARIF exportado para a aba Security do GitHub

### Containerização
- Build multi-stage (builder Alpine → imagem final mínima)
- `docker compose up -d --build` sobe toda a stack (API + PostgreSQL) em um comando
- Healthcheck no banco de dados garante que a API só inicia quando o PostgreSQL está pronto
- Migrations executadas automaticamente na inicialização da aplicação

### Observabilidade
- Logs estruturados com `log/slog` (nível INFO/WARN/ERROR) em todas as operações relevantes
- Campos contextuais nos logs: `id`, `email`, `status`, `error`
- Job Summary no GitHub Actions com tabelas de vulnerabilidades por scanner

### Testabilidade
- Interfaces de repositório permitem mocks sem banco real
- Cobertura de testes: ~82% nos use cases, ~96% nos value objects
- Testes unitários independentes de infraestrutura (sem banco, sem HTTP)

### Qualidade de código
- Arquitetura API-first: contrato OpenAPI é a fonte da verdade
- Código de roteamento e serialização gerado — sem drift entre contrato e implementação
- Clean Architecture: regras de negócio sem dependência de framework ou banco

### Portabilidade
- Configuração 100% via variáveis de ambiente (sem config files)
- Sem dependência de SO específico; compila e roda em Linux, macOS e Windows

---

## Documentação adicional

| Documento | Conteúdo |
|-----------|----------|
| [docs/ubiquitous-language.md](ubiquitous-language.md) | Glossário dos termos do domínio |
| [docs/ddd-aggregates.md](ddd-aggregates.md) | Agregados, entidades internas e invariantes |
| [docs/bounded-contexts.md](bounded-contexts.md) | Contextos delimitados e mapa de contextos |
| [docs/architecture-decisions.md](architecture-decisions.md) | ADRs: PostgreSQL, chi, oapi-codegen |
| [docs/openapi.yaml](openapi.yaml) | Contrato OpenAPI completo da API |
| [docs/postman_collection.json](postman_collection.json) | Coleção Postman com todos os endpoints |
