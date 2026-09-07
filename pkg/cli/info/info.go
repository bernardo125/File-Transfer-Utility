package info

import "fmt"

// PrintMenu displays the main menu options.
func PrintMenu() {
	fmt.Println("=== File Transfer Utility ===")
	fmt.Println("[1] Send file")
	fmt.Println("[2] Receive file")
	fmt.Println("[3] Exit")
}

// PrintMyToken displays the user's signaling token.
func PrintMyToken(token string) {
	fmt.Printf("\nYOUR TOKEN:\n%s\n\n", token)
}

// PrintExecutingHolePunch displays status when starting UDP hole punching.
func PrintExecutingHolePunch() {
	fmt.Println("Executing UDP Hole Punching...")
}

// PrintHolePunchSuccess displays success after UDP hole punching succeeds.
func PrintHolePunchSuccess() {
	fmt.Printf("Hole Punch successful")
}

// PrintConnectingQUIC displays status when establishing the QUIC connection.
func PrintConnectingQUIC() {
	fmt.Println("Establishing QUIC connection...")
}

// PrintConnectionSuccess displays success when QUIC connection is established.
func PrintConnectionSuccess() {
	fmt.Println("Connection established successfully!")
}

// PrintSendSuccess displays a success message when file transfer and verification succeed.
func PrintSendSuccess() {
	fmt.Println("\n[+] Transfer and integrity verification completed successfully!")
}

// PrintWaitingForTransfer displays status when waiting for incoming peer transfer requests.
func PrintWaitingForTransfer() {
	fmt.Println("\n[*] Waiting for transfer requests from the peer...")
}

// PrintWaitingForNextFile displays status when waiting for the next file transfer.
func PrintWaitingForNextFile() {
	fmt.Println("\n[*] Waiting for the next file...")
}

// PrintTransferSuccess displays a success message when a file has been downloaded and verified.
func PrintTransferSuccess(fileName string) {
	fmt.Printf("\n[+] File '%s' downloaded and verified successfully!\n", fileName)
}
