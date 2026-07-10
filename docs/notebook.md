# Transferência de ficheiros pela internet entre peers
    Este projeto permite a transferencia de dados sem limite entre peers na internet. Isto surge como uma alternativa aos metodos tradicionais , como por whatsapp, discord que tem limite de dados para ser enviados ou google drive que transferencias muito lentas. O objetivo desta utility é ter algo simples e que envie dados de forma rápida (Gbs)

## O problema das NAT
NAT (netword address translation) é um mecanismo devido á falta de end. ipv4 que permite que um host privado tenha um endereço que esteja publicamente "escondido" por de trás do endereço do router. Para tal, no router há uma tabela de tradução que contém o endereço privado do host e o endereço público do host na internet, pelo que quando há comunicação o router traduz e reencaminha os pacotes para o respetivo host
Para que a comunicação entre dois peers seja possivel, começamos logo á partida com um problema. Como é que os dois peers podem se conectar numa internet com falta de end. ipv4 e que os dispositivos estao atrás de uma NAT?

A solução para isso é através do protocolo STUN (session travessal utilities for nat) que é possível enviar um pacote de pedido de endereço público a um server público , na internet, um server de STUN, que vai analisar o pacote recebido e responder com o respetivo endereço público. Assim o host pode saber o seu endereço público, trocar esta info com o outro peer e ser possível a comunicação

**O objetivo desta utility é nunca depender de qualquer tipo de servidor externo (para além do stun que já há gratuitos públicos), daí o servidor de signaling (intermediador incial enquanto a comunicação entre os peers nao é estabelecida) normalmente usado neste tipo de arquiteturas vai ser omitido.**

## A SOLUCAO: KEY TROCADA ENTRE PEERS
Na verdade este será encriptado (por exemplo por AES-GCM), juntamente com outras informações como palavra passe da sessao, e vai passar por uma função de hash na base64 para obter uma key simples.

Esta key (token) é fundamental para a comunicação. Cada peer deverá enviar esta key ao outro peer, por um outro qualquer tipo de meio de comunicação seguro (tipo whatsapp, discord - neste caso vai ser tipo um server de "signaling" manual) e esta key permitirá os peer poderem se encontrar. Isto irá permitir reduzir infraestrutura (basta os peers e o server de STUN), através de um código simples tal como uma chave de um lobby de uma sala de um jogo online.

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
Para garantir segurança e para não haver alterações na chave enviada entre os peers optou-se por encriptar a chave através de AES-GCM. O motivo do uso deste mecanismo é devido á sua alta compatibilidade, garante boa segurança e deteta se houve alterações nos dados (autenticidade). Funciona por encriptação simétrica, o que é essencial para esta aplicação (assim os dois peers podem obter os valores não encriptados). 
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