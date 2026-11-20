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
		fmt.Print("Choose a option: ")

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
		fmt.Print("Type your friend's token: ")
		token, err := reader.ReadString('\n')
		if err != nil {
			// input closed (EOF): retrying would spin forever
			return "", err
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
	fmt.Print("\nType the path of the file you want to send (or 'exit' to close): ")
	filePath, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	filePath = strings.TrimSpace(filePath)
	// for windows copy paste compatibility
	if strings.HasPrefix(filePath, "\"") {
		filePath, _ = strings.CutPrefix(filePath, "\"")
		filePath, _ = strings.CutSuffix(filePath, "\"")
	}

	return filePath, nil
}

// AskSendAnother prompts the user whether they want to send another file.
func AskSendAnother(reader *bufio.Reader) (bool, error) {
	fmt.Print("\nDo you want to send other? (Y/N): ")
	ans, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	ans = strings.TrimSpace(strings.ToUpper(ans))
	return ans == "Y", nil
}

// AskAcceptTransfer prompts the receiver whether to accept or reject an incoming file transfer.
func AskAcceptTransfer(reader *bufio.Reader, fileName string, fileSize int64) (bool, error) {
	fmt.Printf("\n[?] Peer wants to send %q (%.2f MB). Accept? (Y/N): ",
		fileName, float64(fileSize)/(1024*1024))

	ans, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	ans = strings.TrimSpace(strings.ToUpper(ans))
	return ans == "Y", nil
}

// AskContinueReceiving prompts the receiver whether to stay in the session waiting for more files.
func AskContinueReceiving(reader *bufio.Reader) (bool, error) {
	fmt.Print("\nDo you want to wait to receive more files? (Y/N): ")
	ans, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	ans = strings.TrimSpace(strings.ToUpper(ans))
	return ans == "Y", nil
}
