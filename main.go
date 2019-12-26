package main

import (
  "fmt"
  "bufio"
  "strings"
  "os"
  "net"
  "encoding/binary"
  "crypto/aes"
  "crypto/cipher"
)

func exit(err string) {
  fmt.Println(err)
  os.Exit(0)
}

func sendFile(client string, port string, file string, cp cipher.Block) {
  f, err := os.Open("./" + file)
  if err != nil {
    exit("unable to open file: " + file)
  }
  defer f.Close()

  // ACCEPT CLIENT
  ln, err := net.Listen("tcp6", ":" + port)
  if err != nil {
    exit("unable to listen on port 12345")
  }

  var s net.Conn
  for {
    s, err = ln.Accept()
    if err != nil {
      continue
    }
    rclient, _, err := net.SplitHostPort(s.RemoteAddr().String())
    if err != nil {
      exit("unable to parse remote client addr")
    }
    if rclient == client {
      break
    } else {
      fmt.Println("invalid client tried to connect:", rclient)
    }
  }
  defer s.Close()

  fmt.Println("Client [", s.RemoteAddr().String(), "] successfully connected")

  // SERVER HELLO
  var fileSize int64
  fi, err := f.Stat()
  if err != nil {
    exit("unable to get stat of file")
  }
  sizeB := make([]byte, 8)
  binary.LittleEndian.PutUint64(sizeB, uint64(fi.Size()))
  _, err = s.Write(sizeB)
  if err != nil {
    exit("unable to send file size")
  }
  fileSize = fi.Size()

  // SEND FILE
  readWrite(func (b []byte) (int, error) {
    return f.Read(b)
  }, func (b []byte) {
    cp.Encrypt(b, b)
  }, func (b []byte, length int) (int, error) {
    return s.Write(b)
  }, cp.BlockSize(), fileSize)

  fmt.Println("Done sending file")
}

func recvFile(host string, port string, file string, cp cipher.Block) {
  f, err := os.Create("./" + file)
  if err != nil {
    exit("unable to open file: " + file)
  }
  defer f.Close()

  // CONNECCT TO SERVER
  s, err := net.Dial("tcp6", net.JoinHostPort(host, port))
  if err != nil {
    exit("unable to listen on port 12345")
  }
  defer s.Close()

  // CLIENT HELLO
  var fileSize int64
  sizeB := make([]byte, 8)
  l, err := s.Read(sizeB)
  if err != nil || l != 8 {
    exit("unable to receive file size")
  }
  fileSize = int64(binary.LittleEndian.Uint64(sizeB))

  fmt.Println("Successfully connected to host [", s.RemoteAddr().String(), "]")

  // RECEIVE DATA
  readWrite(func (b []byte) (int, error) {
    return s.Read(b)
  }, func(b []byte) {
    cp.Decrypt(b, b)
  },func (b []byte, length int) (int, error) {
    return f.Write(b[:length])
  }, cp.BlockSize(), fileSize)

  fmt.Println("Done receiving file")
}

func readWrite(read func(b []byte) (int, error),
    mmap func(b []byte),
    write func(b []byte, length int) (int, error),
    blockSize int, size int64) {
  var bytes int64 = 0
  data := make([]byte, blockSize)
  for {
    for i := range data {
      data[i] = 0
    }

    length, err := read(data)
    if err != nil {
      exit("unable to read")
    }
    mmap(data)
    _, err = write(data, length)
    if err != nil {
      exit("unable to write")
    }
    bytes += int64(length)
    if bytes >= size {
      bytes = size
    }
    fmt.Printf("\033[2K\r%d / %d", bytes, size)
    if bytes == size {
      break
    }
  }
  fmt.Println()
}

func printUsage() {
  fmt.Println("usage: fs [s|r] [file] [ip] [port]")
}

func read(reader *bufio.Reader) string {
  res, err := reader.ReadString('\n')
  if err != nil {
    exit("unable to read from stdin")
  }
  res = strings.Replace(res, "\n", "", -1)
  return res
}

func main() {
  fmt.Println(" ----- GO FS ----- ")

  reader := bufio.NewReader(os.Stdin)

  var sr   string
  var host string
  var port string
  var file string
  var key  string

  fmt.Print("Send or Receive [s|r]: ")
  sr = read(reader)

  fmt.Print("Host (IPv6): ")
  host = read(reader)

  fmt.Print("Port: ")
  port = read(reader)

  fmt.Print("File: ")
  file = read(reader)

  fmt.Print("Key (32 bytes): ")
  key = read(reader)
  if len(key) != 32 {
    exit("invalid key length")
  }

  cp, err := aes.NewCipher([]byte(key))
  if err != nil {
    exit("unable to construct cipher from key")
  }

  fmt.Println("Block Size", cp.BlockSize())

  switch sr {
  case "s":
    fmt.Println("Sending", file, "to", host)
    sendFile(host, port, file, cp)
  case "r":
    fmt.Println("Receiving", file, "from", host)
    recvFile(host, port, file, cp)
  default:
    printUsage()
  }

  fmt.Println(" ----- ----- ----- ")
}
