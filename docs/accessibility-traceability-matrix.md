# Acessibilidade — Matriz de rastreabilidade (EN 301 549 / WCAG 2.1 AA)

Autoavaliação da interface do FTU (File Throw) feita sobre o código da branch `feature/ui-redesign`.

> **Âmbito e limites.** Este documento é uma **autoavaliação técnica**, não uma certificação nem um parecer jurídico de conformidade com a Diretiva (UE) 2019/882 (Ato Europeu da Acessibilidade). Usa a norma harmonizada **EN 301 549 V3.2.1** como referência, que remete para as **WCAG 2.1 nível AA**. O FTU é uma aplicação de secretária (Wails: Go + webview), por isso aplica-se a cláusula 11 (software) — cujos requisitos de conteúdo coincidem com as WCAG — e as cláusulas 5 (genérica) e 12 (documentação). Onde a evidência é só "inspeção de código" ou onde nada foi testado, está dito explicitamente.

## 1. Como foi avaliado

| Método | Detalhe | Reprodução |
|---|---|---|
| Teste automático axe-core | axe-core 4.14.0, tags `wcag2a, wcag2aa, wcag21a, wcag21aa, wcag22aa, best-practice`; 8 estados × 2 temas (home, pairing, connecting, transfer, awaiting, transferring, completed, deciding) | `frontend/scripts/a11y-audit.cjs` |
| Testes dirigidos (19) | títulos, foco após navegação, ordem de tabulação, anel de foco, fluxo só com teclado, movimento reduzido (definição do SO), reflow a 320 px, texto a 200 %, estrutura (h1/main/nomes), anúncio assertivo | idem |
| Lighthouse 13.5.0 | Chromium 156, preset desktop, build de produção (`vite preview`), ecrã Home, tema claro e escuro | `lighthouse http://localhost:4173 --preset=desktop` |
| Cálculo de contraste | Fórmula WCAG 2.x sobre os tokens de `frontend/src/index.css` (tabela na secção 4) | — |
| Inspeção de código | Para critérios sem teste automático (semântica, ordem DOM, ausência de gestos, etc.) | — |

Os testes correm contra o *mock* de desenvolvimento (`frontend/src/services/devMock.ts`) num Chromium headless; **não correm no webview nativo do Wails nem com leitores de ecrã reais** (ver secção 6).

```bash
cd frontend && npm run dev -- --port 5173 &
PUPPETEER=<caminho>/puppeteer-core AXE=<caminho>/axe.min.js CHROME=/usr/bin/chromium \
  node scripts/a11y-audit.cjs resultado.json
```

(`puppeteer-core` e `axe-core` não são dependências do projeto; o pacote global `lighthouse` traz ambos.)

## 2. Resultado resumido

| Indicador | Antes | Depois |
|---|---|---|
| Violações axe (16 estado/tema) | 4 regras em 13 dos 16 estados: `color-contrast`, `page-has-heading-one`, `region`, `definition-list` | **0** |
| Testes dirigidos | n/a (não existiam) | **19/19 passam** |
| Lighthouse Accessibility (claro / escuro) | 95 (primeira versão do redesign, Home, 1 falha de contraste) | **100 / 100** |
| Lighthouse Performance / Best practices | — | 100 / 100 (LCP 0,6 s, TBT 0 ms, CLS 0) |
| Itens "a rever" do axe | — | 4 (etiquetas "You/Peer" sobre fundo liso; axe não consegue medir por haver imagens próximas) — verificados à mão, ver secção 4 |

## 3. Achados e correções

| ID | Achado (antes) | Critério | Correção | Ficheiros |
|---|---|---|---|---|
| F-01 | Texto secundário de 11 px com contraste 4,38:1 (claro) | 1.4.3 | Tokens `--c-dim`/`--c-muted` reforçados | `index.css` |
| F-02 | Idem 4,47:1 e 3,58:1 (escuro, sobre linha ativa) | 1.4.3 | Idem | `index.css` |
| F-03 | Ecrã de Pairing sem `h1` | 1.3.1 / 2.4.6 | `h1` oculto visualmente com o título do ecrã | `PairingScreen.tsx` |
| F-04 | Controlos do topo fora de qualquer landmark | 1.3.1 / 2.4.1 | `<header>` (controlos do ecrã) e `<aside aria-label>` (ferramentas) | `AppHeader.tsx`, `FloatingTools.tsx` |
| F-05 | `<dl>` com filho `<span>` inválido | 1.3.1 / 4.1.1 | Renderização condicional sem elemento órfão | `TransferDetails.tsx` |
| F-06 | Campo do token sem indicador de foco (`outline-none`, só mudava a cor da borda 1 px) | 2.4.7 | Removido `focus:outline-none`; usa o anel global de 2 px | `TokenPanel.tsx`, `index.css` |
| F-07 | Todas as vistas com o mesmo título de documento | 2.4.2 | `useDocumentTitle` por ecrã | `useDocumentTitle.ts`, `App.tsx` |
| F-08 | Foco ficava no elemento removido ao mudar de ecrã | 2.4.3 | Foco programático no contentor do novo ecrã | `App.tsx` |
| F-09 | "Copiado" só visual | 4.1.3 | Região `role="status"` | `TokenPanel.tsx` |
| F-10 | Oferta de ficheiro (exige ação) anunciada só em modo educado | 4.1.3 | `aria-live="assertive"` em `deciding` e `failed` | `TransferScreen.tsx` |
| F-11 | Pedido de confirmação ao desligar durante a transferência só visual | 4.1.3 / 3.3.4 | Mensagem de estado anunciada | `TransferScreen.tsx` |
| F-12 | Animações em ciclo (pássaro no pairing, rasto de bits na transferência) | 2.2.2 / 11.7 | Todas param com `prefers-reduced-motion` (regra `*` em `index.css`) e só correm enquanto há um estado real a representar. **Não há controlo dentro da aplicação** (o botão foi removido a pedido) | `index.css` |
| F-13 | Bordas de campos e da pista de progresso com < 3:1 | 1.4.11 | Token `--c-control` (≥ 3:1) | `index.css`, `TokenPanel.tsx`, `ProgressBar.tsx` |
| F-14 | Tamanhos de letra em `px` | 1.4.4 | Convertidos para `rem` | todos os `.tsx` |
| F-15 | Lista da transferência feita de `div`s sem semântica de tabela | 1.3.1 | `role=table/row/columnheader/cell`; colunas empilham em janelas estreitas | `TransferDetails.tsx` |
| F-16 | A 320 px: chip do cabeçalho sobrepunha o botão Back; botões fixos tapavam o fim do conteúdo | 1.4.10 / 2.4.11 | Chip oculto abaixo de `sm`; espaço inferior reservado | `PairingScreen.tsx`, `HomeScreen.tsx`, `TransferScreen.tsx` |
| F-17 | Rótulo branco da barra medido contra a pista clara (axe) | 1.4.3 | Camada preenchida passa a ter o seu próprio fundo (branco sobre `--c-primary` 4,76:1; falha usa `--c-danger-fill` 6,54:1) | `ProgressBar.tsx` |
| F-18 | Sem suporte a cores forçadas (preenchimento da barra desaparecia) | 11.7 / 1.4.11 | `@media (forced-colors: active)` | `index.css` |

## 4. Contraste calculado (tokens)

| Par | Mínimo | Escuro | Claro |
|---|---|---|---|
| Texto base (`fg` / `bg`) | 4,5:1 | 14,62 | 15,28 |
| Texto secundário (`muted` / `surface`) | 4,5:1 | 6,86 | 8,77 |
| Texto de apoio (`dim` / `surface`) | 4,5:1 | 5,62 | 6,46 |
| Etiquetas "You/Peer" (`dim` / `bg`) | 4,5:1 | 6,12 | 5,57 |
| Texto de apoio sobre linha ativa (accent 14 %/10 % sobre `surface`) | 4,5:1 | 4,51 | 5,68 |
| Texto de destaque (`accent` / `surface`) | 4,5:1 | 5,76 | 5,34 |
| Texto de erro (`danger` / `surface`) | 4,5:1 | 5,23 | 5,56 |
| Branco sobre preenchimento primário (botões, barra) | 4,5:1 | 4,76 | 4,76 |
| Branco sobre preenchimento de erro (barra) | 4,5:1 | 6,54 | 6,54 |
| Borda de campo vs `inset` (componente de UI) | 3:1 | 4,06 | 3,23 |
| Borda de campo vs `surface` | 3:1 | 3,50 | 3,41 |
| Anel de foco (`accent`) vs `bg` | 3:1 | 6,27 | 4,61 |

A margem mínima (4,51:1) é estreita; alterar `--c-dim` ou `--c-accent-soft` exige repetir a auditoria.

## 5. Matriz de rastreabilidade

Estado: **✔ A** = verificado por teste automático · **✔ I** = verificado por inspeção de código · **◐** = parcial · **✖** = não testado · **N/A** = não aplicável.
Testes referem-se a `scripts/a11y-audit.cjs` (axe = execução axe-core).

### 5.1 Perceptível

| ID | EN 301 549 | WCAG | Requisito | Achado | Implementação | Verificação | Estado |
|---|---|---|---|---|---|---|---|
| R-01 | 11.1.1.1 | 1.1.1 | Alternativas textuais a conteúdo não textual | — | Imagens decorativas com `alt=""`; ícones `aria-hidden`; botões só com ícone têm `aria-label`; voo e rasto de bits `aria-hidden` (o estado é dito em texto) | axe `image-alt`, `button-name`; teste "estrutura" (botões sem nome = 0) | ✔ A |
| R-02 | 11.1.2.x | 1.2.x | Média temporizada | — | A aplicação não tem áudio nem vídeo | — | N/A |
| R-03 | 11.1.3.1 | 1.3.1 | Informação e relações | F-03, F-04, F-05, F-15 | `h1` por ecrã, landmarks (`header`, `main`, `aside`), tabela ARIA, `dl` válido, `ol` nas etapas de ligação | axe (0 violações); teste "estrutura" (1 `h1`, 1 `main`) | ✔ A |
| R-04 | 11.1.3.2 | 1.3.2 | Sequência com significado | — | Ordem do DOM = ordem visual (controlos do ecrã → conteúdo → ferramentas) | Inspeção; teste de ordem de tabulação | ✔ I |
| R-05 | 11.1.3.4 | 1.3.4 | Orientação | — | Sem bloqueio de orientação | Inspeção | ✔ I |
| R-06 | 11.1.3.5 | 1.3.5 | Finalidade do campo | — | O único campo é um token técnico (`autocomplete="off"`); não é dado pessoal | Inspeção | N/A |
| R-07 | 11.1.4.1 | 1.4.1 | Utilização da cor | — | Estados dizem-se em texto (Sending, Failed, Declined…), além da cor; ponto de ligação acompanha texto | Inspeção | ✔ I |
| R-08 | 11.1.4.3 | 1.4.3 | Contraste (mínimo) 4,5:1 | F-01, F-02, F-17 | Tokens reforçados; rótulo da barra com fundo próprio | axe `color-contrast` = 0 violações nos 16 estados/temas; secção 4; 4 itens "a rever" calculados à mão | ✔ A |
| R-09 | 11.1.4.4 | 1.4.4 | Redimensionar texto (200 %) | F-14 | Tamanhos em `rem`; conteúdo faz scroll vertical | Teste "texto a 200 %": sem scroll horizontal, ação alcançável | ✔ A |
| R-10 | 11.1.4.5 | 1.4.5 | Imagens de texto | — | O nome "File Throw" é texto; as imagens não contêm texto utilizável | Inspeção | ✔ I |
| R-11 | 11.1.4.10 | 1.4.10 | Reflow (320 px) | F-16 | Colunas da lista empilham; chip do cabeçalho oculto abaixo de `sm`; espaço inferior reservado | Teste "reflow 320 px" em 5 ecrãs (sem scroll horizontal, sem elementos fora do ecrã) | ✔ A |
| R-12 | 11.1.4.11 | 1.4.11 | Contraste de componentes não textuais 3:1 | F-13, F-18 | `--c-control` em campos e pista; anel de foco; regra `forced-colors` | Secção 4 (cálculo). Cores forçadas **não** testadas num SO real | ◐ |
| R-13 | 11.1.4.12 | 1.4.12 | Espaçamento do texto | — | Sem alturas fixas em contentores de texto; linhas quebram | Inspeção; **não** testado com folha de estilo de utilizador | ◐ |
| R-14 | 11.1.4.13 | 1.4.13 | Conteúdo ao pairar/focar | — | Só dicas nativas `title`; sem *popovers* personalizados | Inspeção | N/A |

### 5.2 Operável

| ID | EN 301 549 | WCAG | Requisito | Achado | Implementação | Verificação | Estado |
|---|---|---|---|---|---|---|---|
| R-15 | 11.2.1.1 | 2.1.1 | Teclado | — | Todos os controlos são `<button>`/`<textarea>` nativos; Enter no campo do token liga | Teste "fluxo só com teclado" (Home → Pairing → Transferência) | ✔ A |
| R-16 | 11.2.1.2 | 2.1.2 | Sem armadilha de teclado | — | Sem *focus traps* nem diálogos modais | Inspeção; teste de tabulação | ✔ I |
| R-17 | 11.2.1.4 | 2.1.4 | Atalhos de tecla única | — | Não existem atalhos | Inspeção | N/A |
| R-18 | 11.2.2.1 | 2.2.1 | Tempo ajustável | — | Existem limites: 30 s para ligar e 120 s para aceitar (backend). São de **tempo real** (negociação NAT/protocolo) → exceção "tempo real/essencial" | Inspeção; ver secção 6 | ◐ |
| R-19 | 11.2.2.2 | 2.2.2 | Pausar, parar, ocultar | F-12 | Só a preferência do SO. Animações em ciclo só existem durante estados reais (ligação, bytes a fluir). Sem mecanismo próprio na aplicação: quem não tiver a definição do SO ativa não consegue parar o movimento | Testes "SO reduz movimento" (pairing e rasto de bits) | ◐ |
| R-20 | 11.2.3.1 | 2.3.1 | Três flashes | — | Os dígitos do rasto alternam entre ~4 e ~11 Hz (0,09–0,23 s por fase) mas cada um ocupa uns 3×5 píxeis, bem abaixo da área-limite de flash (estimativa por inspeção, sem medição com ferramenta de flashes) | Análise de área (inspeção) | ◐ |
| R-21 | 11.2.4.1 | 2.4.1 | Contornar blocos | F-04 | Landmarks; vista única com poucos controlos | axe `region` = 0 | ✔ A |
| R-22 | 11.2.4.2 | 2.4.2 | Página com título | F-07 | `document.title` distinto por ecrã | Teste "títulos distintos" (3/3) | ✔ A |
| R-23 | 11.2.4.3 | 2.4.3 | Ordem do foco | F-08 | Foco movido para o novo ecrã; ordem = DOM | Teste "foco move-se para o novo ecrã" | ✔ A |
| R-24 | 11.2.4.4 | 2.4.4 | Finalidade da ligação | — | Não há hiperligações; o botão GitHub tem nome acessível | Inspeção | ✔ I |
| R-25 | 11.2.4.6 | 2.4.6 | Cabeçalhos e etiquetas | F-03 | `h1` descritivo por ecrã; campos com `<label>` | axe `label`, teste "estrutura" | ✔ A |
| R-26 | 11.2.4.7 | 2.4.7 | Foco visível | F-06 | Anel global de 2 px (`:focus-visible`), incluindo o campo de texto | Testes "foco visível em todas as paragens (home)" e "anel no campo" | ✔ A |
| R-27 | 11.2.5.1–4 | 2.5.1–2.5.4 | Gestos, cancelamento, rótulo no nome, movimento | — | Sem gestos nem sensores; ações no *click*/Enter; texto visível faz parte do nome acessível | Inspeção | ✔ I |
| R-28 | — | 2.5.8 (WCAG 2.2) | Tamanho do alvo ≥ 24 px | — | Controlos ≥ 28 px de altura (botões `sm`), ícones 40 px | Inspeção | ✔ I |
| R-29 | — | 2.4.11 (WCAG 2.2) | Foco não ocultado | F-16 | As ferramentas fixas não tapam conteúdo: o ecrã reserva espaço inferior | Teste de reflow; **não** testado com todos os tamanhos de janela | ◐ |

### 5.3 Compreensível

| ID | EN 301 549 | WCAG | Requisito | Achado | Implementação | Verificação | Estado |
|---|---|---|---|---|---|---|---|
| R-30 | 11.3.1.1 | 3.1.1 | Idioma da página | — | `<html lang="en">`; interface em inglês | Teste "estrutura" (`lang`) | ✔ A |
| R-31 | 11.3.1.2 | 3.1.2 | Idioma das partes | — | Mensagens de erro do backend usadas na UI estão em inglês; o *mock* de desenvolvimento tem textos em português (não vai para produção) | Inspeção (`pkg/ui/*.go`) | ◐ |
| R-32 | 11.3.2.1–2 | 3.2.1, 3.2.2 | Sem mudança de contexto ao focar/introduzir | — | Navegação só por ação explícita (botão/Enter) | Inspeção | ✔ I |
| R-33 | 11.3.2.3–4 | 3.2.3, 3.2.4 | Navegação e identificação consistentes | — | Controlos de ecrã e ferramentas partilhados (`AppHeader`, `FloatingTools`, `Button`) | Inspeção | ✔ I |
| R-34 | 11.3.3.1 | 3.3.1 | Identificação do erro | — | Erros em `Notice` com `role="alert"`, em texto | Inspeção; axe `aria-allowed-role` | ✔ I |
| R-35 | 11.3.3.2 | 3.3.2 | Etiquetas ou instruções | — | Cada campo tem etiqueta e instrução ("Send it to your peer…") | Inspeção | ✔ I |
| R-36 | 11.3.3.3 | 3.3.3 | Sugestão de correção | — | Erro de tempo esgotado indica o que verificar | Inspeção | ◐ |
| R-37 | 11.3.3.4 | 3.3.4 | Prevenção de erros | F-11 | Desligar a meio da transferência pede confirmação (4 s); a oferta recebida exige Aceitar/Recusar explícito, sem foco automático | Inspeção | ✔ I |
| R-38 | — | 3.3.7 / 3.3.8 (WCAG 2.2) | Entrada redundante / autenticação acessível | — | Colar o token é permitido e há botão "Paste"; sem testes cognitivos | Inspeção | ✔ I |

### 5.4 Robusto e requisitos específicos de software

| ID | EN 301 549 | WCAG | Requisito | Achado | Implementação | Verificação | Estado |
|---|---|---|---|---|---|---|---|
| R-39 | 11.4.1.1 | 4.1.1 | Análise sintática | F-05 | HTML válido (React; `dl` corrigido) | axe | ✔ A |
| R-40 | 11.4.1.2 | 4.1.2 | Nome, função, valor | F-15 | `progressbar` com `aria-valuenow/valuetext`; toggle com `aria-pressed`; tabela com papéis | axe; testes de estrutura e de botão de movimento | ✔ A |
| R-41 | 11.4.1.3 | 4.1.3 | Mensagens de estado | F-09, F-10, F-11 | Região da fase da transferência (educada, assertiva se exige ação); `status` para copiar, ligação e confirmação; `alert` para erros | Teste "oferta anunciada assertivamente"; inspeção das restantes | ◐ |
| R-42 | 11.5.2.x | — | Interoperabilidade com tecnologias de apoio | — | Semântica HTML/ARIA padrão, exposta pelo webview do SO | **Não testado** com NVDA, Narrator, Orca ou VoiceOver | ✖ |
| R-43 | 11.7 | — | Preferências do utilizador | F-12, F-18 | Tema inicial segue `prefers-color-scheme`; movimento segue `prefers-reduced-motion`; `forced-colors` tratado | Teste de movimento reduzido ✔ A; tema/cores forçadas por inspeção | ◐ |
| R-44 | 5.2 / 5.5 | — | Ativação de funcionalidades; peças operáveis | — | Funcionalidades de acessibilidade ativas por omissão; nada exige agarrar/torcer/forçar | Inspeção | ✔ I |
| R-45 | 12.1.1–2 | — | Documentação acessível | — | Este documento em Markdown (texto), com estrutura de cabeçalhos | Inspeção | ✔ I |
| R-46 | 12.2 | — | Serviços de apoio | — | Projeto de código aberto; apoio via *issues* do repositório | — | N/A |

## 6. Itens em aberto e limitações

1. **Pausa de animações (R-19).** Removido o botão "Reduce animations": o critério 2.2.2 depende agora só da definição do sistema operativo. Se for preciso cumpri-lo sem essa definição, basta reintroduzir um interruptor que ative a mesma regra CSS.
2. **Leitores de ecrã reais (R-42, R-41).** Nenhum teste com NVDA/Narrator (WebView2), Orca (WebKitGTK/AT-SPI) ou VoiceOver (WKWebView). É o passo mais importante antes de afirmar conformidade.
3. **Webview nativo.** Os testes correram em Chromium, não no webview do Wails; comportamento de foco e de `aria-live` pode diferir.
4. **Cores forçadas / Alto Contraste do Windows (R-12, R-43).** Regra CSS adicionada, mas não verificada num SO com esse modo ativo.
5. **Espaçamento de texto (R-13).** Não testado com folha de estilo de utilizador (1.4.12).
6. **Seletor de ficheiros e pasta (nativo).** É fornecido pelo SO (`SelectFileNative`/`OpenFolder`); a sua acessibilidade não depende do FTU.
7. **Barra de título nativa.** O título da janela (`pkg/ui/run.go`: "File Transfer Utility") é estático; o título dinâmico aplica-se ao documento web.
8. **Limites de tempo (R-18).** 30 s para ligar e 120 s para aceitar são impostos pelo protocolo/NAT; estão justificados como tempo real, mas não são ajustáveis. Se vier a ser necessário, o limite de 30 s pode tornar-se configurável sem tocar na segurança.
9. **Margem de contraste.** O par `--c-dim` sobre a linha ativa no tema escuro tem 4,51:1 (mínimo 4,5:1).
10. **Teste com utilizadores.** Não realizado; a conformidade técnica não garante usabilidade para pessoas com deficiência.

## 7. Manutenção

- Antes de cada PR que mexa na UI: `npx tsc -b`, `npm run lint`, `npm run build` e `node scripts/a11y-audit.cjs` (esperado: **0 violações, 19/19 testes**).
- Cores novas entram como tokens em `index.css` e devem ser acrescentadas à tabela da secção 4.
- Qualquer animação nova em ciclo tem de parar com `prefers-reduced-motion` (regra `*` em `index.css`) e só correr enquanto houver um estado real a representar.
- Texto novo usa `rem` (nunca `px`) e controlos novos usam `Button`/`IconButton`, que já trazem foco visível, alvo mínimo e estados desativados.
