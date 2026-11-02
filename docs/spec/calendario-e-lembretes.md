# Calendário e lembretes

Fontes: [eventos Go](../../internal/app/calendar.go), [serviço desktop](../../studio/calendar.go), [calendário](../../studio/frontend/src/CalendarView.svelte), [formulário](../../studio/frontend/src/EventForm.svelte), [repetições no frontend](../../studio/frontend/src/calendar.ts), [feriados](../../studio/frontend/src/holidays.ts), [API de lembretes](../../cloud/worker/reminders.ts) e [regras](../../cloud/worker/reminder-rules.ts). Conferido em 02/11/2026.

## Eventos são notas

A tela Calendário pertence aos cofres `kind = "notes"`. Cada evento é um Markdown direto em `Agenda/` ou `Calendar/`, sem banco local de eventos separado. O corpo guarda as anotações, e o nome do arquivo fornece o título. O leitor consulta ambas as pastas; novas notas usam `Agenda` em português/espanhol ou `Calendar` em inglês, conforme o idioma atual do Studio.

```yaml
---
date: "2026-11-05"
time: "14:00"
event: appointment
repeat: weekly
remind: 1h
---
Anotações do compromisso.
```

| Propriedade | Valores e efeito |
| --- | --- |
| `date` | Data obrigatória `AAAA-MM-DD`; aceita data YAML ou string válida |
| `time` | `HH:MM`; ausente significa dia inteiro |
| `event` | `appointment`, `birthday`, `exam`, `interview`, `deadline`, `reminder`; valor desconhecido lido cai em `reminder` |
| `repeat` | Vazio, `weekly`, `monthly`, `yearly` |
| `remind` | Vazio para não avisar; `0`, `15m`, `1h`, `1d`, `1w` |

`SaveEvent` valida campos, evita colisões com números no nome, mantém o corpo e altera propriedades. Renomear o título usa `RenameItem` e sua correção de links. A criação grava `created` e `updated`; campos opcionais vazios são removidos. A exclusão do formulário usa a lixeira e oferece desfazer. A nota pode ser aberta para escrever e editada também por outro editor.

`Occurrences` no Go e `occurrences` no frontend calculam ocorrências no intervalo solicitado. Repetições mensais/anuais limitam o dia ao último disponível: dia 31 em mês mais curto, ou 29 de fevereiro em outro ano. Horários usam o fuso do computador. Sem horário, a base do aviso é 09:00; `1d` e `1w` recuam por dias de calendário, e minutos/horas por duração.

## Tela

`CalendarView.svelte` mostra seis semanas, dia selecionado, eventos desse dia e os próximos 30 dias a partir de hoje. Domingo inicia a semana em português/inglês; segunda em espanhol. Há setas, Hoje e seletor aberto pelo nome do mês, com os doze meses e navegação por ano. Clique escolhe o dia; duplo clique ou Enter abre criação. Aniversário novo sugere repetição anual; o formulário sugere aviso de um dia.

O dia mostra até três eventos e a quantidade restante; a lateral mostra a lista completa. Nota do dia abre hoje ou um arquivo de outro dia já existente; eventos também aparecem no topo da nota do dia no workspace. Feriados são calculados localmente para Brasil, Estados Unidos ou Espanha conforme o idioma da interface. Não são notas, não geram avisos e não são uma consulta a calendário oficial por localização. A escolha de exibi-los fica em `localStorage["blup.cal.holidays"]`.

## Avisos no computador

O serviço de notificações do Wails é iniciado por `Studio.ServiceStartup`. Se estiver disponível, uma rotina verifica a cada 30 segundos todos os cofres conhecidos em `config.Global`, mesmo sem serem o cofre aberto, enquanto o aplicativo funciona, inclusive na bandeja. Não existe daemon de agenda instalado separadamente.

O aviso local mostra título e indicação do dia/horário; no Linux usa `<cache>/blup/notify-icon.png` como imagem. O macOS pode pedir autorização ao salvar evento com aviso. Clicar traz a janela e emite `calendarOpen`; o workspace navega ao calendário quando o cofre corresponde ao atual, sem prometer troca automática para outro cofre.

Ocorrências vencidas há até 12 horas ainda podem avisar quando o Studio volta. Os IDs enviados são guardados em `<config>/blup/reminders-sent.json`, modo `0600`, para evitar repetição após reinício. Esse registro contém caminho local do cofre, da nota, dia e antecedência; não é enviado pela API de lembretes. Entradas com mais de 31 dias são retiradas ao salvar o registro. A leitura sempre devolve um mapa utilizável; JSON `null`, tempos inválidos e entradas danificadas não causam escrita em mapa nulo.

`notifications` em `studio-prefs.json` é `true` por padrão. `SetNotifications` salva a escolha e solicita nova rodada. Desligar impede avisos locais e solicita lista vazia para cancelar e-mails pendentes; sem rede, o cancelamento remoto ainda depende de comunicação posterior. Falha do serviço de notificações não impede abrir o Studio. O cancelamento remoto tem repetição própria a cada cinco minutos enquanto os avisos ficam desligados, inclusive quando o loop de notificações não iniciou. Sair envia lista vazia antes de encerrar a sessão; sem comunicação, guarda o trabalho em `signout-pending` para repetir na inicialização ou próximo login.

## Avisos por e-mail

Max permite 50 e Ultra 100 envios aceitos pelo provedor por mês UTC; Grátis e Pro não têm esse recurso. O e-mail genérico diz que existe um lembrete e orienta abrir o Studio. Não contém título, tipo, corpo, links de notas ou data detalhada do compromisso. O logo vem de `static/email/blup.png`. As notas da agenda podem viajar pelo sync cifrado, mas seu texto não é enviado em claro por `/api/reminders`.

`pushMail` prepara até 300 instantes dos cofres conhecidos para aproximadamente os próximos 31 dias, com tolerância de 12 horas passadas. `app.ReminderID` usa HMAC-SHA-256 de diretório local, nota, dia e antecedência, truncado nos primeiros 16 bytes e codificado em hexadecimal. A chave aleatória de 32 bytes fica em `<config>/blup/reminder-key`, modo `0600`; não viaja para o servidor. Se não puder persistir a chave, usa uma só da execução, podendo mudar IDs depois de reiniciar. Não é um ID global do evento entre computadores. O cliente ordena a lista e envia quando muda ou a cada seis horas. Planos com limite zero são consultados novamente somente após seis horas; falhas têm espera de cinco minutos. A chamada tem timeout de 20 segundos.

| API autenticada | Efeito |
| --- | --- |
| `GET /api/reminders` | Limite do plano e contador usado no mês UTC |
| `PUT /api/reminders` | Substitui pendentes da conta com `{reminders: [{id, at}]}`; `at` é Unix em milissegundos |

O schema aceita no máximo 300 IDs únicos, de 32 a 64 caracteres hexadecimais, e horários inteiros positivos. `keep` limita a janela a 12 horas de atraso e 32 dias adiante. Listas recebidas substituem os pendentes da conta; ocorrências já marcadas não são reinseridas. Ao receber lista num plano sem e-mails, apaga pendentes. Não há fila separada do D1 nem agendamento no provedor para cada evento.

`sendDue`, disparado pelo cron `*/5 * * * *`, lê até 100 vencidos por rodada. Reivindica cada item por lease de 120 segundos e reserva uma vaga mensal considerando `sent + reserved`, por operações condicionais D1. A tentativa é contada antes da chamada; há no máximo três por lembrete. Até 50 reivindicações vencidas são recuperadas por rodada, permitindo continuar depois de uma queda.

Se o provedor aceita o envio, confirma o contador mensal e marca `sentAt`. Em falha, libera a reserva e espera pelo menos cinco minutos para tentar novamente; ao esgotar três tentativas, marca como pulado. Atraso superior a 12 horas ou plano sem e-mails também pula o item. Cota mensal esgotada impede nova reivindicação, sem ultrapassar o limite; o item pode acabar expirando pela janela de atraso.

A lista substitui apenas itens ainda não reivindicados. Cancelar preferências ou sair não desfaz um envio que já está em andamento. Horários já deixados na fila podem ser executados com o Studio fechado. A limpeza diária remove itens sem lease com mais de 36 horas e contadores antigos sem reserva, em até 20 lotes de cem; atrasos de limpeza continuam na rodada seguinte.

Aceitação pelo Email Service não comprova entrega à caixa de entrada. Uma queda após o envio e antes de persistir a confirmação pode repetir a mensagem; as três tentativas limitam o trabalho, sem garantir exatamente uma entrega. A lista por conta e os IDs derivados de caminhos locais também não deduplicam universalmente várias instalações. [Segurança](../seguranca.md) descreve esses limites, e [deploy](../deploy.md) trata a configuração do serviço.
