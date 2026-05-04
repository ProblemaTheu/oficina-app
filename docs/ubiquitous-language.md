# Linguagem Ubíqua — Tech Challenge

> Glossário dos termos do domínio utilizados em todo o código, banco de dados, API e documentação.
> Qualquer novo desenvolvimento deve utilizar estes termos sem tradução ou sinônimos.

---

## Fluxo Principal

| Termo | Descrição |
|-------|-----------|
| **Ordem de Serviço (OS)** | Entidade central do sistema. Registra a solicitação de reparo ou manutenção de um veículo, desde a abertura até a entrega. Identificada por um número único gerado automaticamente (ex: `OS-000042`). |
| **Diagnóstico** | Análise técnica realizada pelo mecânico após a OS ser recebida. Descreve o problema identificado no veículo e embasa o orçamento. |
| **Orçamento** | Resultado do diagnóstico: lista de serviços e peças com valores. Enviado ao cliente para aprovação antes da execução. |
| **Aprovação** | Aceite formal do cliente ao orçamento apresentado. Autoriza o início da execução dos serviços. |
| **Reprovação** | Recusa do cliente ao orçamento. Encerra a OS sem execução. |
| **Execução** | Etapa em que o mecânico realiza efetivamente os serviços aprovados e consome as peças do estoque. |
| **Entrega** | Ato de devolver o veículo ao cliente após a conclusão dos serviços. Encerra o ciclo da OS. |

---

## Atores

| Termo | Descrição |
|-------|-----------|
| **Atendente** | Usuário responsável pelo contato com o cliente: abre OS, registra entregas e consulta status. |
| **Mecânico** | Usuário responsável pela execução técnica: realiza diagnóstico, avança status e finaliza serviços. |
| **Administrador** | Usuário com acesso completo ao sistema, incluindo cadastro de serviços, peças e usuários. |

---

## Entidades de Cadastro

| Termo | Descrição |
|-------|-----------|
| **Cliente** | Pessoa física ou jurídica proprietária de um ou mais veículos. Identificado por CPF (11 dígitos) ou CNPJ (14 dígitos). |
| **Veículo** | Automóvel pertencente a um cliente. Identificado pela placa (formato antigo `ABC1234` ou Mercosul `ABC1D23`). |
| **Placa** | Value Object que representa a identificação única de um veículo. Validada no momento do cadastro. |
| **Serviço** | Atividade técnica oferecida pela oficina (ex: "Troca de óleo"). Possui preço base e tempo estimado em minutos. |
| **Peça / Insumo** | Material físico utilizado durante a execução de um serviço (ex: "Pastilha de freio dianteira"). Possui código interno, preço e controle de estoque. |

---

## Estoque

| Termo | Descrição |
|-------|-----------|
| **Estoque Atual** | Quantidade disponível de uma peça no momento presente. |
| **Estoque Mínimo** | Quantidade mínima aceitável de uma peça. Serve de alerta para reposição. |
| **Entrada** | Ajuste de estoque que aumenta a quantidade disponível (recebimento de mercadoria). |
| **Saída** | Ajuste de estoque que diminui a quantidade disponível (baixa manual). |
| **Ajuste** | Correção manual da quantidade em estoque, para mais ou para menos, sem vinculação a uma OS. |
| **Dedução automática** | Baixa de estoque de peças realizada automaticamente pelo sistema quando a OS avança para `em_execucao`. |

---

## Máquina de Estados da OS

| Termo | Descrição |
|-------|-----------|
| **Status** | Estado atual de uma Ordem de Serviço dentro do seu ciclo de vida. |
| **Transição** | Mudança de um status para outro. Somente transições válidas são permitidas pelo domínio. |
| **Histórico de Status** | Registro imutável de cada transição realizada numa OS, com o status anterior, o novo status, timestamp e observação opcional. |

### Estados

| Status | Significado |
|--------|-------------|
| `recebida` | OS aberta; aguardando atribuição a um mecânico. |
| `em_diagnostico` | Mecânico analisando o veículo. |
| `aguardando_aprovacao` | Orçamento enviado; aguardando decisão do cliente. |
| `em_execucao` | Serviços em andamento (estoque deduzido automaticamente). |
| `finalizada` | Serviços concluídos; aguardando entrega do veículo. |
| `entregue` | Veículo devolvido ao cliente. Estado terminal. |

---

## Segurança

| Termo | Descrição |
|-------|-----------|
| **Usuário** | Conta de acesso ao sistema. Possui nome, e-mail, senha (armazenada como hash bcrypt) e papel. |
| **Papel (Role)** | Define as permissões de um usuário: `mecanico`, `atendente` ou `administrador`. |
| **JWT** | Token de autenticação emitido no login. Válido por 8 horas. Deve ser enviado no header `Authorization: Bearer <token>`. |
