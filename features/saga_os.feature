# language: pt
Funcionalidade: Ciclo de vida da ordem de serviço coordenado por saga
  Para atender clientes de várias filiais com segurança
  Como oficina
  Quero que abertura, orçamento, pagamento e execução se completem ou se desfaçam juntos

  Os passos falam só HTTP com os três serviços (OS, Billing, Execution) e com
  o stub do Mercado Pago — nenhum passo lê banco, a mesma regra dos serviços.

  Contexto:
    Dado uma peça com 5 unidades em estoque

  Cenário: OS orçada, paga e executada
    Dado um cliente ativo com um veículo
    Quando o atendente abre uma OS com 2 unidades da peça
    Então a OS fica "recebida" e 2 unidades da peça ficam reservadas
    Quando o mecânico conclui o diagnóstico "troca do filtro"
    Então eventualmente a OS fica "aguardando_aprovacao" com um orçamento gerado
    Quando o cliente aprova o orçamento
    E o Mercado Pago aprova o pagamento
    Então eventualmente a OS fica "em_execucao"
    Quando o mecânico inicia e conclui o reparo
    Então eventualmente a OS fica "finalizada"
    E a peça tem 3 unidades e nenhuma reservada
    E o pagamento do orçamento está "aprovado"

  Cenário: falha no reparo depois do pagamento estorna e desfaz tudo
    Dado um cliente ativo com um veículo de placa de falha simulada
    Quando o atendente abre uma OS com 2 unidades da peça
    E o mecânico conclui o diagnóstico "troca do filtro"
    E eventualmente a OS fica "aguardando_aprovacao" com um orçamento gerado
    E o cliente aprova o orçamento
    E o Mercado Pago aprova o pagamento
    Então eventualmente a OS fica "cancelada"
    E eventualmente o pagamento do orçamento está "estornado"
    E a cobrança consta como estornada no Mercado Pago
    E a peça tem 5 unidades e nenhuma reservada

  Cenário: cliente recusa o orçamento
    Dado um cliente ativo com um veículo
    Quando o atendente abre uma OS com 2 unidades da peça
    E o mecânico conclui o diagnóstico "troca do filtro"
    E eventualmente a OS fica "aguardando_aprovacao" com um orçamento gerado
    E o cliente recusa o orçamento
    Então eventualmente a OS fica "cancelada"
    E eventualmente a peça tem 5 unidades e nenhuma reservada

  Cenário: cartão recusado e nova tentativa no mesmo link bloqueada
    Dado um cliente ativo com um veículo
    Quando o atendente abre uma OS com 2 unidades da peça
    E o mecânico conclui o diagnóstico "troca do filtro"
    E eventualmente a OS fica "aguardando_aprovacao" com um orçamento gerado
    E o cliente aprova o orçamento
    E o Mercado Pago recusa o pagamento
    Então eventualmente a OS fica "cancelada"
    E eventualmente a peça tem 5 unidades e nenhuma reservada
    E uma nova tentativa de pagamento no mesmo link é recusada
