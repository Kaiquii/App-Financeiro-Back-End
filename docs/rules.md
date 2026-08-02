# Regras Do Projeto

## VM E Producao

- Todo trabalho deve ser realizado somente no projeto local por padrao.
- Nunca conectar na VM, enviar arquivos, executar comandos remotos, alterar configuracoes, manipular containers, acessar o banco de producao, reiniciar a API ou realizar deploy sem autorizacao explicita do usuario para aquela operacao.
- O fornecimento de IP, chave SSH, credenciais ou acesso anterior nao representa autorizacao permanente.
- Pedidos como `implementar`, `corrigir`, `alinhar`, `testar`, `verificar` ou `pode fazer` autorizam apenas alteracoes locais, salvo quando o usuario mencionar explicitamente que autoriza a acao na VM ou em producao.
- Antes de qualquer acao na VM, explicar exatamente o que sera executado, quais servicos podem ser afetados e qual sera o procedimento de rollback. Aguardar confirmacao explicita antes de continuar.
- Toda autorizacao para a VM e limitada ao escopo informado e termina quando aquela operacao for concluida. Uma nova alteracao ou um novo deploy exige uma nova autorizacao.
- Nunca interpretar uma autorizacao para backup, auditoria, migration ou consulta como permissao para publicar uma imagem, recriar containers ou alterar a API em producao.
- Depois de concluir alteracoes locais, entregar os comandos ou o procedimento para o usuario realizar o deploy manualmente, a menos que ele autorize explicitamente o deploy pela ferramenta.
- Em caso de duvida sobre o alcance da autorizacao, parar e pedir confirmacao. Nao executar a acao remota por inferencia.
