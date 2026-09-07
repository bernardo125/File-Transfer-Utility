package errors

import "fmt"

// PrintHolePunchFailure displays an error message when UDP hole punching fails.
func PrintHolePunchFailure(err error) {
	fmt.Printf("Falha no Hole Punch: %v\n", err)
}

// PrintConnectionError displays an error when connection fails.
func PrintConnectionError(err error) {
	fmt.Printf("Erro na ligação: %v\n", err)
}

// PrintInvalidOption displays a warning when the user chooses an invalid menu option.
func PrintInvalidOption() {
	fmt.Println("Opção inválida.")
}

// PrintInvalidToken displays an error message when an invalid token is entered.
func PrintInvalidToken(err error) {
	fmt.Printf("Token inválido: %v\n", err)
}

// PrintExecutionError displays an error that occurred during execution.
func PrintExecutionError(err error) {
	fmt.Printf("Erro durante a execução: %v\n", err)
}

// PrintSendFailure displays a failure message during file sending.
func PrintSendFailure(err error) {
	fmt.Printf("[-] Falha no envio: %v\n", err)
}

// PrintTransferError displays an error message during file reception.
func PrintTransferError(err error) {
	fmt.Printf("[-] Erro na transferência: %v\n", err)
}
