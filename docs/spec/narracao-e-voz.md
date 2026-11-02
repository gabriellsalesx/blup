# Narração e voz

Fontes: [API de narração](../../internal/app/narration_api.go), [idioma da nota](../../internal/app/translate.go), [serviço Studio](../../studio/narration.go), [texto falável](../../internal/narrate/text.go), [OpenRouter](../../internal/narrate/openrouter.go), `internal/app/narration_blup.go`, `internal/narrate/models.go`, `internal/narrate/mp3.go`, `internal/kokoro/` e `studio/frontend/src/Props.svelte`.

## Fluxo no Studio

A narração é acionada no formulário de propriedades da nota. A condição exata em `Props.svelte` é `!isIndex && (kind === "" || kind === "blog") && !props.blup`: exclui `_index.md` e notas com `blup` preenchido, mas permite notas sem tipo de seção e da seção `blog`. Como `kindOf`, em `studio/notes.go`, só reconhece tipos nas pastas marcadas do Bento, notas Bento fora dessas pastas também podem ser narradas. A API Go pode ser chamada para uma nota por caminho; ela própria valida senha e texto mínimo.

`Studio.Narrate` resolve o idioma com `app.NoteOrigin(site, rel)`, usa a voz salva em `Prefs.Voices[lang]` e chama `app.Narrate`, que resolve novamente o idioma da nota antes de escolher o motor. O progresso sai por eventos `narrate` com `note`, `done` e `total`. A tela mostra preparação, partes e salvamento. A instalação local emite `voiceInstall` com bytes e etapa.

O resultado tem `link`, `hash`, `seconds`, `bytes` e `used`. O MP3 já foi gravado em anexos, mas as propriedades da nota são atualizadas pela interface, que envia edições de `audio`, `audio_hash` e `audio_seconds` ao salvamento normal. `app.Narrate` não grava sozinho essas propriedades.

O link tem formato `[[Nome.mp3]]`. Uma narração anterior apontando para MP3 simples existente na pasta de anexos é reutilizada e sobrescrita. Sem ela, o nome vem da nota, com sanitização e número para evitar colisão; se o idioma da nota difere do principal, acrescenta ` (en)`, ` (es)` ou ` (pt)` antes da extensão. A pasta acompanha o idioma principal do site: `Anexos` para `pt`, `Adjuntos` para `es` ou `Attachments` para inglês e fallback. O arquivo máximo é 25 MiB, mesmo quando o provedor de publicação tem limite diferente.

Uma nota cuja propriedade `password` tenha valor não vazio e diferente de texto `false` é recusada pela API de narração. Isso evita gerar um arquivo público de áudio com o corpo protegido. A regra não apaga MP3 já criado antes de a senha ser configurada.

## Preparação do texto

`narrationText` lê a nota do disco e combina título, descrição e corpo. O título usa propriedade `title`, senão o nome do arquivo sem extensão. Não usa o mesmo fallback de primeiro H1 do partial do tema.

`narrate.Speakable` remove comentários Obsidian e HTML, blocos de código cercados, fórmulas, imagens e embeds, regras, linhas de tabela reconhecidas e referências de notas de rodapé `[^n]`. Remove tags HTML mantendo seu texto, além dos atributos `{.classe}` e `{#id}`. Nas definições de nota de rodapé, descarta apenas a primeira linha `[^n]:`; linhas de continuação podem permanecer no texto falado. Mantém o texto dos links, aliases, ênfases e código inline sem sua sintaxe. Callouts e citações conservam conteúdo falável; títulos e itens recebem pontuação para pausas. A transformação usa expressões regulares e regras de linha, não um parser Markdown completo.

Parágrafos são separados por linhas em branco. O fingerprint é SHA-256 desse texto, truncado aos primeiros oito bytes e codificado em 16 caracteres hexadecimais. `NarrationHash` permite comparar o texto atual com `audio_hash`; diferença sinaliza áudio antigo, sem regeneração automática.

A API rejeita texto com menos de três espaços. O frontend dispara a narração sobre o arquivo salvo, portanto a sincronização do editor com o disco é relevante ao narrar.

## Idioma da nota e escolha de voz

`NoteOrigin`, em `internal/app/translate.go`, considera a pasta imediatamente acima da nota. Se seu nome exato é `pt`, `en` ou `es` e difere de `site.Lang`, a nota tem esse idioma de tradução; pastas em maiúsculas não entram nesse teste. A rotina procura a original na pasta mãe por `translationKey` e depois pelo nome; mesmo sem encontrar a original, conserva o idioma da pasta e devolve o caminho esperado da original. Fora dessa condição, usa `site.Lang`. Não detecta o idioma pelo texto nem pela linguagem da interface do Studio.

Por exemplo, num site `pt`, `Posts/Texto.md` usa `Prefs.Voices["pt"]`, `Posts/en/Text.md` usa `["en"]` e `Posts/es/Texto.md` usa `["es"]`. Só uma pasta de idioma ancestral mais distante não determina o idioma: o teste é da pasta imediata. As preferências são globais à máquina, em `studio-prefs.json`. Configurações, na aba Narração, oferece Português, English e Español para escolher cada uma.

| Representação | Interpretação |
| --- | --- |
| `kokoro:pf_dora` | Voz local |
| `blup:<modelo>\|<voz>` | Voz remota pelo uso de IA do plano Blup Cloud, de qualquer modelo de voz do OpenRouter que responde MP3 (escolhas antigas `blup:<voz>` valem como do modelo recomendado) |
| `microsoft/mai-voice-2.1\|pt-BR-Luana:MAI-Voice-2.1` | Modelo e voz OpenRouter |
| `<modelo>\|` | Voz padrão do modelo |
| Nome antigo sem `\|` | Voz do modelo remoto recomendado |

`voiceFor(lang, chosen)`, em `internal/app/narration_api.go`, aplica esta ordem:

1. Conserva `kokoro:<id>` se o ID existe e fala o idioma da nota. Essa aceitação não testa a instalação; uma escolha local válida com instalação ausente pode falhar ao gerar.
2. Para `blup:`, preserva modelo e voz dos outros modelos para validação remota. Só no modelo recomendado normaliza a voz pelo idioma com `narrate.VoiceFor`, caindo para a voz padrão desse idioma quando necessário. A autorização e o uso do plano são verificados no fluxo remoto.
3. Conserva uma escolha OpenRouter de outro modelo para validação posterior. No modelo recomendado, só a conserva se `narrate.VoiceFor(lang, voz)` devolver a mesma voz.
4. Sem escolha aplicável, usa a primeira voz Kokoro compatível se o motor estiver instalado; sem ela, usa o modelo remoto recomendado com `narrate.DefaultVoice(lang)`.

Uma voz local incompatível é descartada, e o modelo recomendado resolve o idioma conforme essas regras. Outros modelos conservam a escolha para validação posterior; não se atribui detecção universal de idioma às vozes deles. Para idiomas fora de `pt`, `en` e `es`, as rotinas usam inglês como fallback. O retorno ao padrão resolve escolhas ausentes ou incompatíveis; falha de síntese de uma voz já aceita não inicia automaticamente outro motor.

`NarrationVoices(idioma, modelo, modeloBlup)` lista locais, vozes do modelo escolhido para o Blup Cloud (só modelos pagos: os gratuitos não passam pelo plano) e vozes do modelo OpenRouter solicitado. Nas Configurações, os grupos Blup Cloud e OpenRouter só aparecem depois de conectar; o do Blup Cloud tem o mesmo seletor de modelo, sem preço, com o aviso de que modelos mais caros gastam a cota mais rápido. Informa motor, nome, disponibilidade e existência de amostra. Disponibilidade OpenRouter significa haver chave salva; no Blup Cloud depende de o plano incluir IA. Isso não garante saldo ou sucesso da próxima chamada. Amostras gravadas estão em `studio/frontend/public/voices`. `Studio.NarrationEstimate` usa o mesmo idioma e preferência para decidir se deve estimar o uso de IA do Blup Cloud.

## OpenRouter

O cliente usa a base `https://openrouter.ai/api/v1`, token global em `credentials.toml` e atribuição HTTP de Blup Studio. `ConnectOpenRouter` verifica `GET /key` antes de salvar. `AIConnected` verifica apenas a presença da chave.

O modelo recomendado codificado é `microsoft/mai-voice-2.1`. Os padrões são Luana (`pt-BR`) para português, Harper (`en-US`) para inglês e Valeria (`es-MX`) para espanhol. `Known()` contém vozes e estimativa de preço fallback; não é uma tabela de preços atualizada fora do código.

`Models` consulta `/models?output_modalities=speech`, filtra famílias configuradas como apenas PCM e interpreta vozes, descrição e preço. Usa heurísticas de nome para idioma e de descrição para modelo multilíngue. Sem voz nomeada, pode oferecer a voz padrão. O modelo recomendado vem primeiro; os outros são ordenados por nome na implementação, apesar de comentário antigo mencionar preço.

`NarrationModels` filtra por idioma e expõe `perPost = PerThousand * 7`, uma estimativa para cerca de 7.000 unidades de texto. Valor zero pode representar outra forma de cobrança. Se a consulta falhar ou vier vazia, usa `Known` como fallback e pode retornar também o erro ao frontend.

Antes de narrar remotamente, `resolveOpenRouter` tenta consultar os modelos atuais. Se a voz saiu, usa a primeira compatível do mesmo modelo; se o modelo saiu, tenta o recomendado e depois outro que fale o idioma. Retorna a escolha substituta em `used`. Quando a lista não pode ser consultada, tenta a escolha original. Se nenhum modelo fala o idioma, recomenda a opção local na mensagem implementada.

`Split` usa `MaxPart = 2500`. Prefere cortes entre parágrafos, depois frases e palavras. O limite é calculado com `len(string)`, isto é, bytes UTF-8, apesar do comentário usar caracteres. Uma palavra isolada maior que o limite não é subdividida pela regra atual.

Cada parte é enviada sequencialmente por `POST /audio/speech` com `model`, `input`, `response_format = "mp3"` e voz quando não vazia. O cliente tem timeout padrão de três minutos por requisição. Tenta até três vezes em falhas de rede elegíveis, HTTP 429, HTTP 5xx ou áudio vazio, com espera entre tentativas. Distingue chave inválida, falta de créditos e modelo removido.

`JoinMP3` analisa frames MPEG Layer III, remove tags ID3 e frames informativos Xing/Info/VBRI das partes e junta o áudio sem ferramenta externa. Soma a duração dos frames. Dados sem frames válidos produzem `ErrNotMP3`.

O frontend decide entre reproduzir a amostra gravada indicada por `sample` e chamar `PreviewVoice`. Essa API sintetiza via OpenRouter ou, para escolha `blup:`, pela API de IA do Blup Cloud com a autorização da conta. Exige a credencial correspondente antes mesmo de consultar a amostra em cache. Sintetiza um texto curto, salva em `<cache>/blup/voice-samples/<hash>.mp3` e reutiliza; o hash considera idioma e escolha. `PreviewVoice` não oferece síntese local automática.

## Voz pelo Blup Cloud

`internal/app/narration_blup.go` trata escolhas `blup:<modelo>|<voz>`. `EstimateNarration` conta os caracteres Unicode do texto falável e consulta o preço e o uso disponível. `speakBlup` verifica se o texto inteiro cabe no uso restante, considerando o teto semanal, e envia partes com o modelo e a voz escolhidos. No modelo recomendado, consulta `AIUsage.perCharMicros`. Para os demais, `perCharMicros` exige encontrar preço positivo na lista OpenRouter; preço desconhecido ou falha de consulta interrompe a estimativa e a narração, sem substituir pelo preço do modelo recomendado.

O servidor em [ai.ts](../../cloud/worker/ai.ts) valida modelo/voz e calcula por requisição:

```text
modelo recomendado: perCharMicros = 22
outros modelos: perCharMicros = Number(pricing.prompt) * 1.000.000
custoMicros = ceil(quantidade de caracteres Unicode * perCharMicros)
```

O catálogo dos outros modelos é consultado em `/models?output_modalities=speech`, com cache de até seis horas por isolate. Exclui famílias configuradas como PCM/local, exige preço positivo e aceita somente voz listada, ou vazia quando o modelo não lista vozes. Na falta de catálogo, só resta o recomendado. A requisição de voz admite até 4.000 caracteres Unicode; o cliente normalmente divide em 2.500 bytes. A confirmação do catálogo no Studio e o cache do servidor ocorrem em momentos diferentes, portanto a estimativa não é uma reserva de preço ou cota.

A base mensal é 660.000 milionésimos de dólar multiplicados pelo fator do plano (Pro 1, Max 2, Ultra 5). A janela móvel dos últimos sete dias admite metade da cota mensal, que renova no primeiro dia UTC. Narração e tradução usam a mesma cota. O preço por modelo vem do catálogo e da fórmula implementada; a narração não reconcilia posteriormente `usage.cost` ou uma fatura do provedor. A tradução registra esse custo reportado quando disponível, senão uma estimativa. A verificação de saldo e a gravação posterior de uso não constituem uma reserva atômica entre chamadas concorrentes.

O servidor e o OpenRouter recebem o texto em claro para síntese remota, conforme [Segurança](../seguranca.md). Os MP3 são unidos por `JoinMP3`; falha de crédito, preço ou autorização interrompe, sem troca automática para Kokoro. Controles da conta e uso estão em [Studio](studio.md#configurações-globais), e a tabela dos planos em [Blup Cloud](blup-cloud-sync.md#planos-executados).

## Kokoro local

`internal/kokoro` usa Kokoro 82M v1.0 ONNX quantizado em 8 bits, ONNX Runtime carregado dinamicamente, eSpeak NG para fonemas e LAME para codificar MP3. Depois de instalada, a geração local não envia o texto ao OpenRouter.

| Idioma | Vozes oferecidas, na ordem |
| --- | --- |
| `pt` | Dora `pf_dora`, Alex `pm_alex`, Santa `pm_santa` |
| `en` | Heart `af_heart`, Bella `af_bella`, Michael `am_michael`, Fenrir `am_fenrir`, Emma UK `bf_emma`, George UK `bm_george` |
| `es` | Dora `ef_dora`, Alex `em_alex`, Santa `em_santa` |

`g2p.go` chama eSpeak por partes, preservando pontuação, escolhendo `pt-br`, `en-us`, `en-gb` ou `es`, e converte IPA para o alfabeto de tokens Kokoro. O vocabulário está em `vocab.go`. Tokens desconhecidos são retirados. Partes de fonemas são cortadas em até 510 símbolos, com pads e estilo conforme comprimento.

`Engine.Speak` processa parágrafos, carrega o estilo da voz e executa sessão ONNX com `input_ids`, `style` e `speed`, obtendo `waveform` em 24 kHz. Insere pausa de 0,9 segundo após o primeiro parágrafo, 0,6 entre os demais e nenhuma após o último. Usa LAME em processo separado para MP3 mono 64 kbit/s, com entrada PCM de 16 bits.

`ort_cgo.go` é compilado somente com `cgo && (linux || darwin)`. Usa `dlopen`, `OrtGetApiBase` e a API C vendorizada em `internal/kokoro/ort/`. `ort_other.go` devolve `ErrUnsupported` em Windows e builds sem CGO.

## Instalação local e plataformas

O diretório é `<cache>/blup/kokoro`. `Install` trabalha em `kokoro.part`, baixa e verifica SHA-256 de cada pacote/arquivo, extrai o necessário, grava `ready.json` e promove a instalação. O layout tem `ort/`, `tools/`, `model.onnx` e `voices/`. `Version` do layout é `1`.

`Installed` confere `ready.json`, versão, biblioteca, modelo e executáveis eSpeak/LAME. Não recalcula checksums de toda a instalação em cada consulta e não percorre todas as vozes nessa verificação rápida.

| Plataforma na tabela | Situação executada hoje |
| --- | --- |
| Linux amd64 | Pacotes ONNX e ferramentas configurados; precisa de build com CGO |
| Linux arm64 | ONNX configurado, pacote de ferramentas ausente; `Supported()` falso |
| macOS arm64 | ONNX configurado, pacote de ferramentas ausente; `Supported()` falso |
| Demais | Sem conjunto de downloads suportado |

O tamanho informado soma runtime, ferramentas, modelo de 92.361.116 bytes e as doze vozes da tabela, de 522.240 bytes cada. A tabela atual usa ONNX Runtime `1.31.1`, tag Blup `voice-tools-1` e revisão fixa `1939ad2a8e416c0acfeecc08a694d14ef25f2231` do repositório Hugging Face do modelo.

`BLUP_VOICE_TOOLS` é lida na inicialização do pacote Go e substitui somente o endereço do pacote de ferramentas por URL ou caminho local. Serve como espelho ou cópia local do mesmo arquivo: o SHA-256 e o tamanho usado no progresso continuam fixos na tabela. Um build diferente exige atualizar o hash da tabela e ajustar o tamanho informado; `fetch` verifica o hash, sem uma comparação separada de tamanho. A variável não completa uma plataforma que esteja sem ferramentas na tabela. O comentário que sugere testar um build novo só com essa variável não corresponde à implementação.

`RemoveLocalVoice` remove a instalação do cache. Ele não apaga automaticamente os MP3 já gerados nas notas nem as preferências de escolha de voz.

## Construção das ferramentas

`studio/build/voice/build-tools.sh` baixa eSpeak NG `1.52.0` e LAME `3.100` dos sources oficiais, valida SHA-256 e compila. Em Linux usa link estático. Empacota os executáveis, tabelas fonéticas, os dicionários `pt_dict`, `en_dict` e `es_dict`, a pasta `lang/` inteira do eSpeak, licenças, referências de sources e o próprio script em `blup-voice-tools-<os>-<arch>.tar.gz`.

O script aceita diretório de trabalho e reconhece amd64/arm64 em Linux ou macOS. Essa capacidade de construir um pacote não significa que ele esteja adicionado com hash à tabela de instalação nem publicado pelo workflow atual. A integração do player publicado está descrita em [temas](temas-hugo.md).
