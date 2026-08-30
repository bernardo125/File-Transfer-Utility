package transfer

import (
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

func SendMetadata(metadata []byte, commStream *quic.Stream) error {
	// send file metadata
	log.Println("Sending metadata to other peer")
	_, writeMetaErr := commStream.Write(metadata)
	if writeMetaErr != nil {
		return writeMetaErr
	}

	return nil

}

/*
// recieves a file from whoever establishes a quic connection with the other peer
func receiveFile(addr string, conn *quic.Conn, commStream *quic.Stream) error {
	newFile, err := os.Create(saveLocation + "/" + friend.FileName)
	if err != nil {
		log.Println("Error: " + err.Error())
		notifyFrontEnd(err.Error() + "")
		return err
	}
	defer newFile.Close()

	log.Println("Recieving file!")
	start := time.Now()

	io.Copy(newFile, commStream)

	fmt.Printf("Finished transfer in %.2f seconds!", time.Since(start).Seconds())
	return nil
}*/
