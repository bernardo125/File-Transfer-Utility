package input

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"ftu/pkg/cli/info"
)

// NewReader creates a new buffered reader reading from standard input.
func NewReader() *bufio.Reader {
	return bufio.NewReader(os.Stdin)
}

// AskMenuOption displays the menu and prompts the user to select an option.
func AskMenuOption(reader *bufio.Reader) (string, error) {
	for {
		info.PrintMenu()
		fmt.Print("Escolha uma opção: ")

		option, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		option = strings.TrimSpace(option)

		if option != "1" && option != "2" && option != "3" {
			continue
		}
		return option, nil
	}
}

// AskPeerToken prompts the user to input the peer's token.
func AskPeerToken(reader *bufio.Reader) (string, error) {
	for {
		fmt.Print("Insira o token do outro peer: ")
		token, err := reader.ReadString('\n')
		if err != nil {
			continue
		}

		token = strings.TrimSpace(token)
		if len(token) <= 12 {
			fmt.Println("The token must have more than 12 bytes")
			continue
		}
		return token, nil
	}
}

// AskFilePath prompts the user for the file path to send.
func AskFilePath(reader *bufio.Reader) (string, error) {
	fmt.Print("\nIntroduza o caminho do ficheiro a enviar (ou 'sair' para terminar): ")
	filePath, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// for windows copy paste compatibility
	if strings.HasPrefix(filePath, "\"") {
		filePath, _ = strings.CutPrefix(filePath, "\"")
		filePath, _ = strings.CutSuffix(filePath, "\"")
	}

	return strings.TrimSpace(filePath), nil
}

// AskSendAnother prompts the user whether they want to send another file.
func AskSendAnother(reader *bufio.Reader) (bool, error) {
	fmt.Print("\nDeseja enviar outro ficheiro? (S/N): ")
	ans, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	ans = strings.TrimSpace(strings.ToUpper(ans))
	return ans == "S" || ans == "Y", nil
}

// AskAcceptTransfer prompts the receiver whether to accept or reject an incoming file transfer.
func AskAcceptTransfer(reader *bufio.Reader, fileName string, fileSize int64) (bool, error) {
	fmt.Printf("\n[?] Peer quer enviar o ficheiro '%s' (%.2f MB). Aceitar? (S/N): ",
		fileName, float64(fileSize)/(1024*1024))

	ans, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	ans = strings.TrimSpace(strings.ToUpper(ans))
	return ans == "S" || ans == "Y", nil
}

// AskContinueReceiving prompts the receiver whether to stay in the session waiting for more files.
func AskContinueReceiving(reader *bufio.Reader) (bool, error) {
	fmt.Print("\nDeseja continuar na sessão a aguardar mais ficheiros? (S/N): ")
	ans, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	ans = strings.TrimSpace(strings.ToUpper(ans))
	return ans == "S" || ans == "Y", nil
}
