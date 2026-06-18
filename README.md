# File-Transfer-Utility
## Descrição

Aplicação de linha de comandos para transferência de ficheiros peer-to-peer (P2P) focada no máximo desempenho bruto. Utiliza o protocolo QUIC puro e UDP Hole Punching manual, eliminando o *overhead* de bibliotecas de rede complexas. Assegura velocidades de transferência limitadas apenas pela capacidade física da rede e pelo disco.

## Arquitetura

* **Linguagem:** Go (Goroutines para concorrência de I/O).
* **Camada de Transporte:** `quic-go` (TLS 1.3 integrado, controlo de congestionamento moderno, eliminação de *head-of-line blocking*).
* **Descoberta de Rede:** Cliente STUN para resolução de IP público e porta.
* **Sinalização:** *Out-of-band* (ex: aplicações de mensagens). A string de ligação é encriptada e codificada antes da partilha.
* **NAT Traversal:** UDP Hole Punching nativo instanciando o QUIC sobre um socket UDP estático e partilhado.

## Roadmap de Desenvolvimento

| Fase | Componente | Tarefas Principais |
| --- | --- | --- |
| **1** | **Fundações UDP e STUN** | - Inicializar `net.ListenUDP` numa porta local fixa.- Implementar cliente STUN para extrair IP:Porta do router. - Validar comportamento em redes locais vs. redes externas. |
| **2** | **Criptografia e Chaves** | - Implementar AES-GCM (256-bit) com chave simétrica partilhada/PIN.- Comprimir e codificar o IP:Porta resultante em Base64.- Implementar função reversa (descodificar e desencriptar). |
| **3** | **UDP Hole Punching** | - Desenvolver ciclo de sincronização para envio de pacotes simultâneos.- Forçar a abertura da porta no NAT enviando tráfego cego para o destino. |
| **4** | **Ligação QUIC** | - Injetar o socket UDP perfurado na configuração do `quic-go`.- Estabelecer *handshake* e multiplexação de canais bidirecionais. |
| **5** | **Motor de Transferência** | - Implementar leitura e escrita em disco por blocos (gestão de RAM).- Calcular taxa de transferência em tempo real (MB/s). |

## Fluxo Operacional (Exemplo)

1. Os utilizadores iniciam a aplicação. O programa resolve o IP público via STUN e gera a chave local
2. O envio do ficheiro é preparado no lado do **Peer A**
3. O **Peer A** e o **Peer B** trocam as suas chaves geradas através de um canal externo seguro (Whatsapp, Discord, ...)
4. Ambos introduzem a chave do destinatário no programa
5. O programa inicia o UDP Hole Punching
6. A ligação QUIC é estabelecida e o ficheiro transita à velocidade máxima disponível

## Desempenho e Métricas Alvo

* **Latência de ligação:** 1-RTT (após Hole Punching).
* **Segurança:** Encriptação fim-a-fio inquebrável por design (TLS 1.3 sobre QUIC).
* **Alocação de Memória:** Otimizada para leitura em *chunks*, suportando envio de ficheiros com múltiplos Gigabytes sem sobrecarga de RAM.