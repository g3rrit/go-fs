package main

import (
  "fmt"
  "os"
  "net"
  "encoding/binary"
)

func exit(err string) {
  fmt.Println(err)
  os.Exit(0)
}

func sendFile(client string, port string, file string) {
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
  }, func (b []byte) (int, error) {
    return s.Write(b)
  }, fileSize)

  fmt.Println("Done sending file")
}

func recvFile(host string, port string, file string) {
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
  }, func (b []byte) (int, error) {
    return f.Write(b)
  }, fileSize)

  fmt.Println("Done receiving file")
}

func readWrite(read func(b []byte) (int, error), write func(b []byte) (int, error), size int64) {
  var bytes int64 = 0
  dataSize := 4096
  data := make([]byte, dataSize)
  for {
    length, err := read(data)
    if err != nil {
      exit("unable to read")
    }
    _, err = write(data[:length])
    if err != nil {
      exit("unable to write")
    }
    bytes += int64(length)
    fmt.Printf("\033[2K\r%d / %d", bytes, size)
    if bytes == size {
      break
    }
  }
  fmt.Println()
}

func printUsage() {
  fmt.Println("usage: ./fs [s|r] [file] [ip] [port]")
}

func main() {
  fmt.Println("GO FS")
  args := os.Args[1:]

  if len(args) != 4 {
    printUsage()
    os.Exit(0)
  }

  switch args[0] {
  case "s":
    fmt.Println("Sending", args[1], "to", args[2])
    sendFile(args[2], args[3], args[1])
  case "r":
    fmt.Println("Receiving", args[1], "from", args[2])
    recvFile(args[2], args[3], args[1])
  default:
    printUsage()
  }
}
