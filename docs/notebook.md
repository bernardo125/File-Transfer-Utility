# Transferência de ficheiros pela internet entre peers
    Este projeto permite a transferencia de dados sem limite entre peers na internet. Isto surge como uma alternativa aos metodos tradicionais , como por whatsapp, discord que tem limite de dados para ser enviados ou google drive que transferencias muito lentas. O objetivo desta utility é ter algo simples e que envie dados de forma rápida (Gbs)

## O problema das NAT
NAT (netword address translation) é um mecanismo devido á falta de end. ipv4 que permite que um host privado tenha um endereço que esteja publicamente "escondido" por de trás do endereço do router. Para tal, no router há uma tabela de tradução que contém o endereço privado do host e o endereço público do host na internet, pelo que quando há comunicação o router traduz e reencaminha os pacotes para o respetivo host
Para que a comunicação entre dois peers seja possivel, começamos logo á partida com um problema. Como é que os dois peers podem se conectar numa internet com falta de end. ipv4 e que os dispositivos estao atrás de uma NAT?

A solução para isso é através do protocolo STUN (session travessal utilities for nat) que é possível enviar um pacote de pedido de endereço público a um server público , na internet, um server de STUN, que vai analisar o pacote recebido e responder com o respetivo endereço público. Assim o host pode saber o seu endereço público, trocar esta info com o outro peer e ser possível a comunicação

**O objetivo desta utility é nunca depender de qualquer tipo de servidor externo (para além do stun que já há gratuitos públicos), daí o servidor de signaling (intermediador incial enquanto a comunicação entre os peers nao é estabelecida) normalmente usado neste tipo de arquiteturas vai ser omitido.**

## A SOLUCAO: KEY TROCADA ENTRE PEERS
Na verdade este será encriptado (por exemplo por Chacha20 ou AES-CTR), juntamente com outras informações como palavra passe da sessao, e vai passar por uma função de hash na base64 para obter uma key simples.

Esta key (token) é fundamental para a comunicação. Cada peer deverá enviar esta key ao outro peer, por um outro qualquer tipo de meio de comunicação seguro (tipo whatsapp, discord - neste caso vai ser tipo um server de "signaling" manual) e esta key permitirá os peer poderem se encontrar. Isto irá permitir reduzir infraestrutura (basta os peers e o server de STUN), através de um código simples tal como uma chave de um lobby de uma sala de um jogo online.
estrutura key:

ip + port local (6 bytes)
ip + port public (6 bytes)
TLS-PSK / Token de Hole Punch (16 bytes)
Nonce Chacha20 (12 bytes)

## Protocolo STUN 

Para o peer saber o seu endereço recorre-se a um pacote stun binding request que tem estas características:
- STUN message type: this tells the server what kind of request this is (here it’s a binding request)
- message length: size of extra data attached to the packet (0 here because we send no extra attributes)
- magic cookie: a fixed constant used by STUN so servers can recognize and validate the packet
- transaction id: 12 random bytes used to match the response with the correct request


## STUN resolve tudo?
A resposta é não. O STUN permite resolver muitos problemas, no entanto, no caso de outro tipos de NAT não é suficiente. Como por exemplo numa NAT simétrica cada par ip:porto só serve para cada endpoint que se está a comunicar (há um novo ip:porto para cada endpoint e server diferente), pelo que o STUN não serve de nada aqui. Para resolver isto temos o TURN que faz relay de todos os pedidos e permite a comunicação. 

**No ambito deste projeto, o TURN não será usado, pelo motivo que não há servers de TURN gratuitos e os que há tem largura de banda e dados enviados limitados, o que estraga o objetivo desta utility.**

Por isso, é provável quem esteja nesta situação esta utility não vai funcionar, isto é , cerca de 15%-20% dos utilizadores da internet

## Encriptação da Key (Token)
Para garantir segurança e para não haver alterações na chave enviada entre os peers optou-se por encriptar a chave através de Chacha20. O motivo do uso deste mecanismo é devido á sua alta compatibilidade, garante boa segurança e deteta se houve alterações nos dados (autenticidade). Funciona por encriptação simétrica, o que é essencial para esta aplicação (assim os dois peers podem obter os valores não encriptados). 
Este token irá conter tanto o endereço ip local, como o endereço ip dado pelo servidor de stun, de modo a garantir que caso a conexão local não seja possível então funciona por stun (este processo ocorre em paralelo e o primeiro a estabelecer a conexão é aquele que será usado). Portanto a key começa a ficar extensa e para manter a simplicidade e não ter um token muito extenso, então este é codificado para um array de 12bytes (address local (6 bytes = ip + port), address stun) pelo que este será encriptado e o seu resultado ainda é codificado para base64. Assim, temos uma key curta (54 caracteres) que terá tudo que é necessário, inclusive segurança, para estabelecer a comunicação entre os peers.

## UDP HOLE PUSH
Apartir do momento, onde ambos os peers trocaram entre si por outro canal de comunicação a key (ou token), então agora temos tudo que é necessário para começar a estabelecer a comunicação. No entanto, apesar do conhecimento de ambos os peers do ip do outro, os routers vão continuar a dar drop a todos os pacotes que não foram solicitados por aquele dispositivo. Desta forma, a conexão continua a não ser possível.
Para tal, temos de recorrer a um mecanismo (UDP HOLE PUSH). Basicamente isto funciona com o envio de pacotes UDP iniciais (tentativa de conexão de forma bruta em simultaneo) que fará com que os routers possam fazer o mapping dos endereços de recessão para saber para qual host redirecionar e ser possível começar a enviar dados.

**(Baseado em https://medium.com/@knightkun/udp-hole-punching-from-scratch-in-golang-building-tsuna-448ca66a733b)**
Considere peer1 e peer2. O que acontece ao nível da NAT:
- O peer1 vai enviar um pacote UDP para o peer2. O router do peer1 vai fazer, portanto, o mapeamento NAT para o src e dst ips. Quando o pacote chega ao router do peer2 este como não tem mapeamento é droped e não é recebido pelo peer2.

- Agora o inverso, o peer2 vai enviar um pacote UDP para o peer1. O router do peer2 vai fazer, portanto, o mapeamento NAT para o src e dst ips. Quando o pacote chega ao router do peer1 como anteriormente foi feito o mapeamento então o pacote é recebido pelo peer1.

Após a resposta do peer1 como já está estabelecido então também chega ao peer2 e já é possível a conexão.
O primeiro pacote UDP funciona cmoo um punch a hole na NAT.

**Em toda a comunicação entre os dois peers é aberto um canal de comunicação (socket), pelo que este será o mesmo para as várias ações da aplicação, desde pedido ao server de stun, udp hole punch e envio de dados**

## Protocolo de envio de dados entre os peers
The protocol follows an HTTP-inspired request/response model, adapted for peer-to-peer file transfer over QUIC. Binary messages are used for transfer requests, responses, and metadata, while the file content is transmitted directly as the body of the QUIC stream.
ARCH:
Sender
                         ┌───────────────┐
                         │      IDLE     │
                         └───────┬───────┘
                                 │
                              REQUEST
                                 │
                                 ▼
                         ┌───────────────┐
                         │   REQUESTED   │
                         └───────┬───────┘
                                 │
                         RESPONSE received
                                 │
                    ┌────────────┴────────────┐
                    │                         │
              ACCEPTED / 2xx              REJECTED
                    │                         │
                    ▼                         ▼
             ┌─────────────┐             ┌──────────┐
             │  TRANSFERRING│             │ REJECTED │
             └──────┬──────┘             └──────────┘
                    │
              Stream FIN
                    │
                    ▼
             ┌─────────────┐
             │  VERIFYING  │
             └──────┬──────┘
                    │
              checksum valid
                    │
                    ▼
             ┌─────────────┐
             │  COMPLETED  │
             └─────────────┘


Receiver:
RECEIVE_REQUEST
      │
      ├── REJECT
      │
      └── ACCEPT
            │
            ▼
      RECEIVE_BODY
            │
            ▼
      VERIFY_CHECKSUM
            │
       ┌────┴────┐
       │         │
      OK       ERROR
       │         │
       ▼         ▼
   COMPLETE    FAILED


TEU PROTOCOLO
─────────────────────────────
O que quero transferir?
Aceitas?
Qual é o ficheiro?
Quanto mede?
Qual é o hash?
Terminou?
Cancela.
─────────────────────────────
              ↓
QUIC
─────────────────────────────
Como transportar os bytes?
Como recuperar perdas?
Como controlar congestionamento?
Como multiplexar streams?
─────────────────────────────
              ↓
UDP

## Interface concept
Quero uma interação de humano computador facil de entender. A ideia é ter um menu inicial onde o utilizador escolhe se pretende enviar ou receber um ficheiro,   depois tem o token que vai partilhar para o outro peer. Após a ligação quic ambos entram numa sessão e a minha ideia é aparecer no terminal como dois computadores ligados. Quando quem vai enviar escolhe o ficheiro e e quem     recebe confirma que quer receber aquele ficheiro (com metadados) então a ligação entre os computadores começa a enviar com uma animação de atirar um aviao de um pc para o outro e em baixo uma barra de progresso e velocidade    de transferencia. Após o envio, os peers podem decidir se pretendem continuar a enviar ficheiros ou terminar a comunicação e voltar ao menu inicial (enviar receber). O menu incial também terá um botao para terminar o programa.


**Arquitetura da TUI**

type sessionState int
const (
    stateMenu sessionState = iota
    stateTokenInput
    stateHolePunching
    stateConnected
    stateTransferring
    stateFinished
)

No método Update(msg tea.Msg) (tea.Model, tea.Cmd):

1. No stateMenu: 
    * O utilizador seleciona com as setas: [1] Enviar Ficheiro ou [2] Receber Ficheiro.
2. No stateTokenInput: 
    * Se for Enviar, você gera o token local usando token.CreateKey(conn) e exibe-o na tela para cópia.
    * Se for Receber, exibe um textinput do Bubbles para colar o token do outro peer.
3. No stateHolePunching: 
    * Você dispara a sua função UDPHolePunch como um tea.Cmd. Quando o canal de resultado receber o endereço, você retorna uma mensagem holePunchSuccessMsg ou holePunchFailedMsg para atualizar a tela.
4. No stateConnected: Você renderiza os dois computadores desenhados em ASCII Art e conectados por uma linha contínua.
    * O emissor abre o filepicker do Bubbles para selecionar o ficheiro. Ao selecionar, envia os metadados (nome do ficheiro, tamanho) via QUIC.
    * O receptor vê na tela: "Peer quer enviar 'video.mp4' (1.2 GB). Aceitar? (S/N)".
5. No stateTransferring: 
    * Dispara-se a transferência QUIC em background. O escritor/leitor do QUIC atualiza um contador de bytes transmitidos.
    * Uma rotina em segundo plano calcula a velocidade (bytes por segundo) e despacha atualizações periódicas de progresso para a UI.* A tela exibe a animação do avião de papel cruzando de uma caixa para a outra, com a barra de progresso (bubbles/progress) e a velocidade correndo abaixo.