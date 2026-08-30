package transfer

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/quic-go/quic-go"
)

// Function to send file to other peer (initialization of stream on quic module)
func SendFile(file *os.File, conn *quic.Conn, commStream *quic.Stream) error {
	defer commStream.Close()
	log.Println("Sending data to other peer")
	start := time.Now()
	_, err := io.Copy(commStream, file)
	if err != nil {
		return err
	}
	log.Printf("Finished transfer in %.2f seconds!", time.Since(start).Seconds())
	return nil

}

func SelectFileToSend(fileName string, conn *quic.Conn) (file *os.File, stats os.FileInfo, hash string, err error) {
	// Use os.Open for read-only access.
	file, err = os.Open(fileName)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to open file: %w", err)
	}

	// Ensure the file is closed even if errors occur later.
	// Capture the close error only if no other error occurred.
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close file: %w", cerr)
		}
	}()

	fileStat, statErr := file.Stat()
	if statErr != nil {
		file.Close()
		return nil, nil, "", fmt.Errorf("failed to get file stats: %w", err)
	}

	file.Close()

	// get hash value of file
	hash, err = hashFileSHA256(file)

	return file, fileStat, hash, err // err will be nil unless file.Close() failed

}

func SendMetadata(metadata []byte, commStream *quic.Stream) error {
	// send file metadata
	log.Println("Sending metadata to other peer")
	_, writeMetaErr := commStream.Write(metadata)
	if writeMetaErr != nil {
		return writeMetaErr
	}

	return nil

}

// recieves a file from whoever establishes a quic connection with the other peer
func receiveFile(addr string, conn *quic.Conn, commStream *quic.Stream) error {
	newFile, err := os.Create(addr)
	if err != nil {
		log.Println("Error: " + err.Error())
		return err
	}
	defer newFile.Close()

	log.Println("Recieving file!")
	start := time.Now()

	io.Copy(newFile, commStream)

	log.Printf("Finished transfer in %.2f seconds!", time.Since(start).Seconds())
	return nil
}
