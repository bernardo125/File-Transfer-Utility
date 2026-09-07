package info

import "fmt"

// PrintMenu displays the main menu options.
func PrintMenu() {
	fmt.Println("=== File Transfer Utility ===")
	fmt.Println("[1] Enviar Ficheiro")
	fmt.Println("[2] Receber Ficheiro")
	fmt.Println("[3] Sair")
}

// PrintMyToken displays the user's signaling token.
func PrintMyToken(token string) {
	fmt.Printf("\nYOUR TOKEN:\n%s\n\n", token)
}

// PrintExecutingHolePunch displays status when starting UDP hole punching.
func PrintExecutingHolePunch() {
	fmt.Println("A executar UDP Hole Punching...")
}

// PrintHolePunchSuccess displays success after UDP hole punching succeeds.
func PrintHolePunchSuccess() {
	fmt.Printf("Hole Punch bem-sucedido")
}

// PrintConnectingQUIC displays status when establishing the QUIC connection.
func PrintConnectingQUIC() {
	fmt.Println("A estabelecer ligação QUIC...")
}

// PrintConnectionSuccess displays success when QUIC connection is established.
func PrintConnectionSuccess() {
	fmt.Println("Ligação estabelecida com sucesso!")
}

// PrintSendSuccess displays a success message when file transfer and verification succeed.
func PrintSendSuccess() {
	fmt.Println("\n[+] Transferência e verificação de integridade concluídas com sucesso!")
}

// PrintWaitingForTransfer displays status when waiting for incoming peer transfer requests.
func PrintWaitingForTransfer() {
	fmt.Println("\n[*] À espera de pedidos de transferência do peer...")
}

// PrintWaitingForNextFile displays status when waiting for the next file transfer.
func PrintWaitingForNextFile() {
	fmt.Println("\n[*] À espera do próximo ficheiro...")
}

// PrintTransferSuccess displays a success message when a file has been downloaded and verified.
func PrintTransferSuccess(fileName string) {
	fmt.Printf("\n[+] Ficheiro '%s' descarregado e verificado com sucesso!\n", fileName)
}
